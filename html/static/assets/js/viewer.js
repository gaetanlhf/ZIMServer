let searchTimeout;
let archiveName;
let lastSearchResults = '';
let connectionLostToast = null;
let originalFaviconHref;

function getOrCreateFaviconLink() {
    let link = document.querySelector('link[rel~="icon"]');
    if (!link) {
        link = document.createElement('link');
        link.rel = 'icon';
        document.head.appendChild(link);
    }
    return link;
}

function initFavicon() {
    const link = getOrCreateFaviconLink();
    const archiveIcon = document.getElementById('archiveIcon');
    originalFaviconHref = archiveIcon ? archiveIcon.src : '';
    if (originalFaviconHref && link.href !== originalFaviconHref) {
        link.href = originalFaviconHref;
    }
}

function updateFaviconFromIframe(iframeDoc) {
    const link = getOrCreateFaviconLink();
    const iconNode = iframeDoc.querySelector(
        'link[rel~="icon" i], link[rel="shortcut icon" i], link[rel="apple-touch-icon" i]'
    );
    const href = iconNode && iconNode.href ? iconNode.href : originalFaviconHref;
    if (href && link.href !== href) {
        link.href = href;
    }
}

function init(archive) {
    archiveName = archive;

    applyTheme();

    document.documentElement.classList.add('viewer-mode');

    initFavicon();

    const iframe = document.getElementById('contentFrame');

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

    try {
        const iframeContent = iframe.contentDocument || iframe.contentWindow.document;
        if (iframeContent.readyState !== "complete") {
            showSpinner();
        }
    } catch (e) {
        showSpinner();
    }

    iframe.addEventListener('load', function() {
        hideSpinner();

        try {
            const iframeWin = iframe.contentWindow;
            const iframeDoc = iframeWin.document;

            const archiveTitle = document.querySelector('.archive-name').textContent;
            document.title = iframeDoc.title + ' - ' + archiveTitle;

            updateFaviconFromIframe(iframeDoc);

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
            applyDarkModeToIframe(iframeDoc);

            updateBrowserURL();
        } catch(e) {
            console.log('Cannot access iframe or content is not fully loaded:', e);
            updateBrowserURL();
        }
    });

    document.addEventListener('click', function(e) {
        const searchResults = document.getElementById('searchResults');
        const searchContainer = document.getElementById('searchContainer');

        if (searchContainer && searchResults) {
            if (!searchContainer.contains(e.target) && !searchResults.contains(e.target)) {
                searchResults.classList.remove('active');
            }
        }
    });

    let resizeTimeout;
    window.addEventListener('resize', () => {
        clearTimeout(resizeTimeout);
        resizeTimeout = setTimeout(positionSearchResults, 100);
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
    setInterval(checkArchiveStatus, 5000);

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        applyTheme();
        const iframe = document.getElementById('contentFrame');
        if (iframe && iframe.contentDocument) {
            applyDarkModeToIframe(iframe.contentDocument);
        }
    });
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

    if (searchContainer && searchResults && searchResults.classList.contains('active')) {
        const rect = searchContainer.getBoundingClientRect();
        searchResults.style.top = (rect.bottom + 4) + 'px';
        searchResults.style.left = rect.left + 'px';
        searchResults.style.width = rect.width + 'px';
    }
}

function fixIframeURLs(iframeDoc) {
    try {
        const currentIframeUrl = iframeDoc.defaultView ? iframeDoc.defaultView.location.href : iframeDoc.URL;

        iframeDoc.body.addEventListener('click', function(e) {
            const link = e.target.closest('a');
            if (!link || link.classList.contains('error-btn')) return;

            const hrefAttr = link.getAttribute('href');
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
                if (!link.target || link.target === '_self') {
                    link.target = '_blank';
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
        });
    } catch(e) {
        console.log('Cannot fix iframe URLs:', e);
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

    if (searchInput.value.length >= 2 && lastSearchResults) {
        positionSearchResults();
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
    const archiveInfo = document.getElementById('archiveInfo');
    if (archiveInfo) {
        archiveInfo.classList.add('loading');
    }
}

function hideSpinner() {
    const archiveInfo = document.getElementById('archiveInfo');
    if (archiveInfo) {
        archiveInfo.classList.remove('loading');
    }
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

                if (data.results && data.results.length > 0) {
                    lastSearchResults = data.results.map(result => {
                        const safePath = result.path.replace(/'/g, "\\'");
                        const safeTitle = result.title.replace(/</g, "&lt;").replace(/>/g, "&gt;");
                        return `<div class="search-result-item" onclick="loadPage('${safePath}')">${safeTitle}</div>`;
                    }).join('');
                    resultsDiv.innerHTML = lastSearchResults;
                    positionSearchResults();
                    resultsDiv.classList.add('active');
                } else {
                    const noResultsText = window.i18n && window.i18n.no_results_found ? window.i18n.no_results_found : 'No results found';
                    lastSearchResults = `<div class="search-result-item no-results">${noResultsText}</div>`;
                    resultsDiv.innerHTML = lastSearchResults;
                    positionSearchResults();
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

function checkArchiveStatus() {
    if (!archiveName) return;
    
    fetch('/api/status')
        .then(res => {
            if (connectionLostToast) {
                connectionLostToast.classList.add('fade-out');
                setTimeout(() => {
                    if (connectionLostToast) {
                        connectionLostToast.remove();
                        connectionLostToast = null;
                    }
                }, 300);
            }
            return res.json();
        })
        .then(data => {
            if (!data.archives.includes(archiveName)) {
                window.location.href = '/';
            }
        })
        .catch(err => {
            console.error(err);
            if (!connectionLostToast) {
                showConnectionLostToast();
            }
        });
}

function showConnectionLostToast() {
    const container = document.getElementById('toastContainer');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = 'toast error';
    const message = window.i18n && window.i18n.connection_lost ? window.i18n.connection_lost : 'Connection lost';
    
    toast.innerHTML = `
        <svg class="toast-icon"  xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 16 16">
          <path d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5m.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2"/>
        </svg>
        <span>${message}</span>
    `;
    
    container.appendChild(toast);
    connectionLostToast = toast;
    
    void toast.offsetWidth;
    
    toast.classList.add('show');
}

function applyTheme() {
    const theme = localStorage.getItem('zimserver_theme') || 'auto';
    if (theme === 'auto') {
        document.documentElement.removeAttribute('data-theme');
    } else {
        document.documentElement.setAttribute('data-theme', theme);
    }
}

function applyDarkModeToIframe(iframeDoc) {
    const forceDarkMode = localStorage.getItem('zimserver_force_dark_mode') === 'true';
    const theme = localStorage.getItem('zimserver_theme') || 'auto';
    const isDark = theme === 'dark' || (theme === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches);

    const iframe = document.getElementById('contentFrame');
    const styleId = 'zimserver-dark-mode-style';
    let style = iframeDoc.getElementById(styleId);

    if (forceDarkMode && isDark) {
        let hasDarkMode = false;
        
        if (iframeDoc.querySelector('meta[name="color-scheme"][content*="dark"]')) {
            hasDarkMode = true;
        }
        
        if (!hasDarkMode && iframeDoc.documentElement.style.colorScheme === 'dark') {
            hasDarkMode = true;
        }

        if (!hasDarkMode && iframeDoc.body) {
            const bodyStyle = getComputedStyle(iframeDoc.body);
            const bodyBg = bodyStyle.backgroundColor;
            
            const rgbMatch = bodyBg.match(/^rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([\d.]+))?\)/);
            if (rgbMatch) {
                const r = parseInt(rgbMatch[1]);
                const g = parseInt(rgbMatch[2]);
                const b = parseInt(rgbMatch[3]);
                const a = rgbMatch[4] !== undefined ? parseFloat(rgbMatch[4]) : 1;
                
                if (a < 0.5) {
                    hasDarkMode = false;
                } else {
                    const brightness = (r * 299 + g * 587 + b * 114) / 1000;
                    if (brightness < 128) {
                        hasDarkMode = true;
                    }
                }
            }
        }

        if (!hasDarkMode) {
            if (!style) {
                style = iframeDoc.createElement('style');
                style.id = styleId;
                style.textContent = `
                    html {
                        filter: invert(1) hue-rotate(180deg) !important;
                        background-color: #ffffff !important;
                    }
                    img, video, iframe, canvas, svg {
                        filter: invert(1) hue-rotate(180deg) !important;
                    }
                `;
                iframeDoc.head.appendChild(style);
            }
            if (iframe) iframe.style.backgroundColor = '#000000';
        } else {
            if (style) style.remove();
            if (iframe) iframe.style.backgroundColor = 'transparent';
        }
    } else {
        if (style) style.remove();
        if (iframe) iframe.style.backgroundColor = '#ffffff';
    }
}
