let searchTimeout;
let archiveName;
let lastSearchResults = '';

function init(archive) {
    archiveName = archive;

    document.documentElement.classList.add('viewer-mode');

    const iframe = document.getElementById('contentFrame');
    const spinner = document.getElementById('spinner');

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
            const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
            const archiveTitle = document.querySelector('.archive-name').textContent;
            document.title = iframeDoc.title + ' - ' + archiveTitle;

            iframeDoc.addEventListener('click', function() {
                const searchResults = document.getElementById('searchResults');
                searchResults.classList.remove('active');
            });

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

        if (!searchContainer.contains(e.target) && !searchResults.contains(e.target)) {
            searchResults.classList.remove('active');
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
    searchInput.addEventListener('input', function() {
        updateClearButton();
    });

    positionSearchResults();
}

function updateBrowserURL() {
    try {
        const iframe = document.getElementById('contentFrame');
        if (iframe.contentWindow.location.origin === window.location.origin) {
            const iframePath = iframe.contentWindow.location.pathname;

            const prefix = '/content/' + archiveName + '/';
            if (iframePath.startsWith(prefix)) {
                const path = iframePath.substring(prefix.length);
                const newUrl = '/viewer/' + archiveName + '/' + path;

                if (window.location.pathname !== newUrl) {
                    history.replaceState(null, '', newUrl);
                }
            }
        }
    } catch(e) {
        console.log('Cannot update browser URL:', e);
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
        links.forEach(link => {
            const handleLinkClick = function(e) {
                const href = this.getAttribute('href');
                if (!href) return;

                if (href.startsWith('http') || href.startsWith('https') || href.startsWith('mailto:')) {
                    if (!this.target || this.target === '_self') {
                        this.target = '_blank';
                    }
                    return;
                }

                if (href.startsWith('#') || href.startsWith('javascript:')) {
                    return;
                }

                e.preventDefault();

                let newPath = href;
                if (href.startsWith('/')) {
                    newPath = href.substring(1);
                } else if (href.startsWith('./')) {
                    newPath = href.substring(2);
                } else if (href.startsWith('../')) {
                    const currentPath = window.location.pathname;
                    const pathParts = currentPath.split('/').filter(p => p);
                    let goBack = (href.match(/\.\.\//g) || []).length;
                    newPath = href.replace(/\.\.\//g, '');

                    if (pathParts.length > 3 + goBack) {
                        pathParts.splice(pathParts.length - goBack - 1, goBack);
                        newPath = pathParts.slice(3).join('/') + '/' + newPath;
                    }
                }

                loadPage(newPath);
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

    if (searchInput.value) {
        clearBtn.classList.add('visible');
    } else {
        clearBtn.classList.remove('visible');
    }
}

function showSearchResults() {
    const searchResults = document.getElementById('searchResults');
    const searchInput = document.getElementById('searchInput');

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

    searchInput.value = '';
    clearBtn.classList.remove('visible');
    searchResults.classList.remove('active');
    searchResults.innerHTML = '';
    searchLoading.classList.remove('active');
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
    const iframe = document.getElementById('contentFrame');
    showSpinner();
    iframe.src = '/content/' + archiveName + '/';

    const searchResults = document.getElementById('searchResults');
    searchResults.classList.remove('active');
}

function loadPage(path) {
    if (path.startsWith('/')) {
        path = path.substring(1);
    }

    const iframe = document.getElementById('contentFrame');
    showSpinner();
    iframe.src = '/content/' + archiveName + '/' + path;

    const searchResults = document.getElementById('searchResults');
    searchResults.classList.remove('active');
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

    if (!query || query.length < 2) {
        resultsDiv.classList.remove('active');
        searchLoading.classList.remove('active');
        lastSearchResults = '';
        return;
    }

    searchTimeout = setTimeout(() => {
        clearBtn.classList.remove('visible');
        searchLoading.classList.add('active');

        fetch('/api/' + archiveName + '/search?q=' + encodeURIComponent(query) + '&limit=10')
            .then(res => res.json())
            .then(data => {
                searchLoading.classList.remove('active');
                if (document.getElementById('searchInput').value) {
                    clearBtn.classList.add('visible');
                }

                positionSearchResults();

                if (data.results && data.results.length > 0) {
                    lastSearchResults = data.results.map(result =>
                        `<div class="search-result-item" onclick="loadPage('${result.path}')">${result.title}</div>`
                    ).join('');
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
                searchLoading.classList.remove('active');
                if (document.getElementById('searchInput').value) {
                    clearBtn.classList.add('visible');
                }
                resultsDiv.classList.remove('active');
                lastSearchResults = '';
            });
    }, 300);
}