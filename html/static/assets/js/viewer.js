let searchTimeout;
let archiveName;
let lastSearchResults = '';

function init(archive) {
    archiveName = archive;

    document.documentElement.classList.add('viewer-mode');

    const iframe = document.getElementById('contentFrame');
    const spinner = document.getElementById('spinner');

    const pathPrefix = '/viewer/' + archiveName + '/';
    if (window.location.pathname.startsWith(pathPrefix)) {
        if (window.location.search || window.location.hash) {
            const currentSrc = iframe.getAttribute('src');
            if (currentSrc) {
                let newSrc = currentSrc;
                if (window.location.search && !newSrc.includes('?')) {
                    newSrc += window.location.search;
                }
                if (window.location.hash && !newSrc.includes('#')) {
                    newSrc += window.location.hash;
                }
                
                if (newSrc !== currentSrc) {
                    iframe.src = newSrc;
                }
            }
        }
    }

    setTimeout(function() {
        try {
            const iframeContent = iframe.contentDocument || iframe.contentWindow.document;
            if (iframeContent.readyState !== "complete") {
                showSpinner();
            }
        } catch (e) {
            showSpinner();
        }
    }, 300);

    iframe.addEventListener('load', function() {
        hideSpinner();

        try {
            const iframeWin = iframe.contentWindow;
            const iframeDoc = iframeWin.document;

            const archiveTitle = document.querySelector('.archive-name').textContent;
            document.title = iframeDoc.title + ' - ' + archiveTitle;

            iframeDoc.addEventListener('click', function() {
                const searchResults = document.getElementById('searchResults');
                if (searchResults) {
                    searchResults.classList.remove('active');
                }
            });

            const originalPushState = iframeWin.history.pushState;
            const originalReplaceState = iframeWin.history.replaceState;

            iframeWin.history.pushState = function() {
                const result = originalPushState.apply(this, arguments);
                updateBrowserURL();
                return result;
            };

            iframeWin.history.replaceState = function() {
                const result = originalReplaceState.apply(this, arguments);
                updateBrowserURL();
                return result;
            };

            iframeWin.addEventListener('hashchange', updateBrowserURL);
            iframeWin.addEventListener('popstate', updateBrowserURL);

            fixIframeURLs(iframeDoc);

            updateBrowserURL();
        } catch(e) {
            console.log('Cannot access iframe or content is not fully loaded:', e);
            updateBrowserURL();
        }
    });

    document.addEventListener('click', function(e) {
        const searchResults = document.getElementById('searchResults');
        const searchInput = document.getElementById('searchInput');
        const searchContainer = document.getElementById('searchContainer');

        if (searchContainer && searchResults) {
            if (!searchContainer.contains(e.target) && !searchResults.contains(e.target)) {
                searchResults.classList.remove('active');
            }
        }
    });

    setTimeout(updateScrollIndicators, 100);

    const header = document.querySelector('.viewer-header');
    header.addEventListener('scroll', updateScrollIndicators);
    window.addEventListener('resize', () => {
        setTimeout(updateScrollIndicators, 100);
        positionSearchResults();
    });

    const searchInput = document.getElementById('searchInput');
    if (searchInput) {
        searchInput.addEventListener('input', function() {
            updateClearButton();
        });
    }

    positionSearchResults();

    window.addEventListener('popstate', function(event) {
        const path = window.location.pathname;
        const search = window.location.search;
        const hash = window.location.hash;
        
        if (path.startsWith('/viewer/' + archiveName + '/')) {
            const entryPath = path.substring(('/viewer/' + archiveName + '/').length);
            showSpinner();
            setIframeLocation('/content/' + archiveName + '/' + entryPath + search + hash);
        } else if (path.startsWith('/catch')) {
            const urlParams = new URLSearchParams(window.location.search);
            const externalUrl = urlParams.get('url');
            if (externalUrl) {
                showSpinner();
                setIframeLocation('/catch?url=' + encodeURIComponent(externalUrl));
            }
        }
    });

    setInterval(updateBrowserURL, 500);
}

function setIframeLocation(url) {
    const iframe = document.getElementById('contentFrame');
    try {
        iframe.contentWindow.location.replace(url);
    } catch (e) {
        console.log('Error using location.replace, falling back to src:', e);
        iframe.src = url;
    }
}

function updateBrowserURL() {
    try {
        const iframe = document.getElementById('contentFrame');
        if (iframe.contentWindow.location.origin === window.location.origin) {
            const iframeLoc = iframe.contentWindow.location;
            const iframePath = iframeLoc.pathname;

            const prefix = '/content/' + archiveName + '/';
            if (iframePath.startsWith(prefix)) {
                const path = iframePath.substring(prefix.length);
                const newUrl = '/viewer/' + archiveName + '/' + path + iframeLoc.search + iframeLoc.hash;
                const currentUrl = window.location.pathname + window.location.search + window.location.hash;

                if (currentUrl !== newUrl) {
                    history.replaceState(null, '', newUrl);
                }
            } else if (iframePath.startsWith('/catch')) {
                const urlParams = new URLSearchParams(iframeLoc.search);
                const externalUrl = urlParams.get('url');
                if (externalUrl) {
                    const newUrl = '/catch?viewer=' + encodeURIComponent(archiveName) + '&url=' + encodeURIComponent(externalUrl);
                    const currentUrl = window.location.pathname + window.location.search + window.location.hash;
                    if (currentUrl !== newUrl) {
                        history.replaceState(null, '', newUrl);
                    }
                }
            }
        }
    } catch(e) {
    }
}

function positionSearchResults() {
    const searchContainer = document.getElementById('searchContainer');
    const searchResults = document.getElementById('searchResults');

    if (searchContainer && searchResults) {
        const rect = searchContainer.getBoundingClientRect();
        searchResults.style.top = (rect.bottom + 4) + 'px';
        searchResults.style.left = rect.left + 'px';
        searchResults.style.width = rect.width + 'px';
    }
}

function fixIframeURLs(iframeDoc) {
    try {
        const links = iframeDoc.querySelectorAll('a');
        const currentIframeUrl = iframeDoc.defaultView ? iframeDoc.defaultView.location.href : iframeDoc.URL;

        links.forEach(link => {
            if (link.classList.contains('error-btn')) {
                return;
            }

            const handleLinkClick = function(e) {
                const hrefAttr = this.getAttribute('href');
                if (!hrefAttr) return;

                if (hrefAttr.startsWith('http://') || hrefAttr.startsWith('https://')) {
                    e.preventDefault();
                    const encodedUrl = encodeURIComponent(hrefAttr);
                    const newBrowserUrl = '/catch?viewer=' + encodeURIComponent(archiveName) + '&url=' + encodedUrl;
                    history.pushState(null, '', newBrowserUrl);
                    showSpinner();
                    setIframeLocation('/catch?url=' + encodedUrl);
                    return;
                }

                if (hrefAttr.startsWith('mailto:')) {
                    if (!this.target || this.target === '_self') {
                        this.target = '_blank';
                    }
                    return;
                }

                if (hrefAttr.startsWith('#') || hrefAttr.startsWith('javascript:')) {
                    return;
                }

                e.preventDefault();

                try {
                    const urlObj = new URL(hrefAttr, currentIframeUrl);
                    
                    if (urlObj.origin === window.location.origin) {
                        const prefix = '/content/' + archiveName + '/';
                        if (urlObj.pathname.startsWith(prefix)) {
                            let relativePath = urlObj.pathname.substring(prefix.length);
                            relativePath += urlObj.search + urlObj.hash;
                            loadPage(relativePath);
                        } else {
                            window.location.href = urlObj.href;
                        }
                    } else {
                        const encodedUrl = encodeURIComponent(urlObj.href);
                        const newBrowserUrl = '/catch?viewer=' + encodeURIComponent(archiveName) + '&url=' + encodedUrl;
                        history.pushState(null, '', newBrowserUrl);
                        showSpinner();
                        setIframeLocation('/catch?url=' + encodedUrl);
                    }
                } catch (err) {
                    console.error("Error parsing URL:", err);
                    loadPage(hrefAttr);
                }
            };

            link.removeEventListener('click', handleLinkClick);
            link.addEventListener('click', handleLinkClick);
        });
    } catch(e) {
        console.log('Cannot fix iframe URLs:', e);
    }
}

function updateScrollIndicators() {
    const header = document.querySelector('.viewer-header');
    const fadeLeft = document.querySelector('.scroll-fade-left');
    const fadeRight = document.querySelector('.scroll-fade-right');

    if (!header || !fadeLeft || !fadeRight) return;

    const hasOverflow = header.scrollWidth > header.clientWidth;
    const scrollLeft = header.scrollLeft;
    const maxScroll = header.scrollWidth - header.clientWidth;

    const canScrollLeft = scrollLeft > 1;
    const canScrollRight = scrollLeft < maxScroll - 1;

    if (canScrollLeft) {
        fadeLeft.classList.add('visible');
    } else {
        fadeLeft.classList.remove('visible');
    }

    if (canScrollRight && hasOverflow) {
        fadeRight.classList.add('visible');
    } else {
        fadeRight.classList.remove('visible');
    }
}

function updateClearButton() {
    const searchInput = document.getElementById('searchInput');
    const clearBtn = document.getElementById('clearSearchBtn');

    if (searchInput && clearBtn) {
        if (searchInput.value) {
            clearBtn.classList.add('visible');
        } else {
            clearBtn.classList.remove('visible');
        }
    }
}

function showSearchResults() {
    const searchResults = document.getElementById('searchResults');
    const searchInput = document.getElementById('searchInput');

    if (!searchResults || !searchInput) return;

    positionSearchResults();

    if (searchInput.value.length >= 2 && lastSearchResults) {
        searchResults.classList.add('active');
    }
}

function clearSearch() {
    const searchInput = document.getElementById('searchInput');
    const clearBtn = document.getElementById('clearSearchBtn');
    const searchResults = document.getElementById('searchResults');
    const searchLoading = document.getElementById('searchLoading');

    if (searchInput) searchInput.value = '';
    if (clearBtn) clearBtn.classList.remove('visible');
    if (searchResults) {
        searchResults.classList.remove('active');
        searchResults.innerHTML = '';
    }
    if (searchLoading) searchLoading.classList.remove('active');
    lastSearchResults = '';
}

function showSpinner() {
    const spinner = document.getElementById('spinner');
    const iframeContainer = document.querySelector('.iframe-container');

    iframeContainer.style.transition = 'opacity 0.4s';
    iframeContainer.style.opacity = '0.5';
    iframeContainer.style.pointerEvents = 'none';

    spinner.classList.add('active');
}

function hideSpinner() {
    const spinner = document.getElementById('spinner');
    const iframeContainer = document.querySelector('.iframe-container');

    iframeContainer.style.transition = 'opacity 0.4s';
    iframeContainer.style.opacity = '1';
    iframeContainer.style.pointerEvents = 'all';

    spinner.classList.remove('active');
}

function loadHome() {
    loadPage('');
}

function loadPage(path) {
    if (path.startsWith('/')) {
        path = path.substring(1);
    }

    const newUrl = '/viewer/' + archiveName + '/' + path;
    const currentUrl = window.location.pathname + window.location.search + window.location.hash;

    if (currentUrl !== newUrl) {
        history.pushState(null, '', newUrl);
    }

    showSpinner();
    setIframeLocation('/content/' + archiveName + '/' + path);

    const searchResults = document.getElementById('searchResults');
    if (searchResults) {
        searchResults.classList.remove('active');
    }
}

function loadRandom() {
    showSpinner();
    fetch('/api/' + archiveName + '/random')
        .then(res => res.json())
        .then(data => {
            if (data.path) {
                loadPage(data.path);
            }
        })
        .catch(err => {
            console.error('Random error:', err);
            hideSpinner();
        });
}

function searchArticles(query) {
    clearTimeout(searchTimeout);
    updateClearButton();

    const resultsDiv = document.getElementById('searchResults');
    const clearBtn = document.getElementById('clearSearchBtn');
    const searchLoading = document.getElementById('searchLoading');
    const searchInput = document.getElementById('searchInput');

    if (!resultsDiv) return;

    if (!query || query.length < 2) {
        resultsDiv.classList.remove('active');
        if (searchLoading) searchLoading.classList.remove('active');
        lastSearchResults = '';
        return;
    }

    searchTimeout = setTimeout(() => {
        if (clearBtn) clearBtn.classList.remove('visible');
        if (searchLoading) searchLoading.classList.add('active');

        fetch('/api/' + archiveName + '/search?q=' + encodeURIComponent(query) + '&limit=-1')
            .then(res => res.json())
            .then(data => {
                if (searchLoading) searchLoading.classList.remove('active');
                if (searchInput && searchInput.value && clearBtn) {
                    clearBtn.classList.add('visible');
                }

                positionSearchResults();

                if (data.results && data.results.length > 0) {
                    lastSearchResults = data.results.map(result => {
                        const safePath = result.path.replace(/'/g, "\\'");
                        const safeTitle = result.title.replace(/</g, "&lt;").replace(/>/g, "&gt;");
                        return `<div class="search-result-item" onclick="loadPage('${safePath}')">${safeTitle}</div>`;
                    }).join('');
                    resultsDiv.innerHTML = lastSearchResults;
                    resultsDiv.classList.add('active');
                } else {
                    lastSearchResults = '<div class="search-result-item no-results">No results found</div>';
                    resultsDiv.innerHTML = lastSearchResults;
                    resultsDiv.classList.add('active');
                }
            })
            .catch(err => {
                console.error('Search error:', err);
                if (searchLoading) searchLoading.classList.remove('active');
                if (searchInput && searchInput.value && clearBtn) {
                    clearBtn.classList.add('visible');
                }
                resultsDiv.classList.remove('active');
                lastSearchResults = '';
            });
    }, 300);
}
