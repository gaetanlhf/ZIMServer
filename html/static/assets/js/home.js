function clearSearch() {
    document.getElementById('searchBox').value = '';
    filterArchives();
}

function resetFilters() {
    document.getElementById('languageFilter').value = '';
    document.getElementById('categoryFilter').value = '';
    document.getElementById('searchBox').value = '';
    filterArchives();
}

let noResultsTimeout;
let filterTimeout;

function filterArchives() {
    clearTimeout(filterTimeout);
    filterTimeout = setTimeout(() => {
        const language = document.getElementById('languageFilter').value.toLowerCase();
        const category = document.getElementById('categoryFilter').value.toLowerCase();
        const search = document.getElementById('searchBox').value.toLowerCase();
        const cards = document.querySelectorAll('.archive-card');
        const clearBtn = document.getElementById('clearSearch');
        const noResults = document.getElementById('noResults');

        localStorage.setItem('zimserver_filter_language', language);
        localStorage.setItem('zimserver_filter_category', category);
        localStorage.setItem('zimserver_filter_search', search);

        if (search) {
            clearBtn.classList.add('visible');
        } else {
            clearBtn.classList.remove('visible');
        }

        let visibleCount = 0;

        // Use a single loop and avoid layout thrashing
        for (let i = 0; i < cards.length; i++) {
            const card = cards[i];
            const cardLang = card.dataset.language.toLowerCase();
            const cardCategory = card.dataset.category.toLowerCase();
            const cardTitle = card.dataset.title.toLowerCase();
            const cardDesc = card.dataset.description.toLowerCase();

            const matchesLanguage = !language || cardLang === language || cardLang === 'mul';
            const matchesCategory = !category || cardCategory === category;
            const matchesSearch = !search || cardTitle.includes(search) || cardDesc.includes(search);

            if (matchesLanguage && matchesCategory && matchesSearch) {
                card.style.display = 'flex';
                card.classList.remove('fade-out');
                visibleCount++;
            } else {
                card.style.display = 'none';
                card.classList.add('fade-out');
            }
        }

        if (noResultsTimeout) {
            clearTimeout(noResultsTimeout);
        }

        if (visibleCount === 0) {
            noResultsTimeout = setTimeout(() => {
                if (noResults) noResults.style.display = '';
            }, 100);
        } else {
            if (noResults) noResults.style.display = 'none';
        }

        const archiveSingular = window.i18n && window.i18n.archive_singular ? window.i18n.archive_singular : 'archive';
        const archivePlural = window.i18n && window.i18n.archive_plural ? window.i18n.archive_plural : 'archives';

        let useSingularForZero = false;
        if (window.i18n && window.i18n.zero_is_singular) {
            useSingularForZero = window.i18n.zero_is_singular;
        }

        let pluralText;
        if (visibleCount === 0) {
            pluralText = useSingularForZero ? archiveSingular : archivePlural;
        } else if (visibleCount === 1) {
            pluralText = archiveSingular;
        } else {
            pluralText = archivePlural;
        }

        document.getElementById('archiveCount').textContent = visibleCount + ' ' + pluralText;
        
        requestAnimationFrame(updateAllScrollIndicators);
    }, 150);
}

function updateScrollIndicators() {
    const filters = document.querySelector('.filters');
    const fadeLeft = document.querySelector('.scroll-fade-left');
    const fadeRight = document.querySelector('.scroll-fade-right');

    if (!filters || !fadeLeft || !fadeRight) {
        return;
    }

    const hasOverflow = filters.scrollWidth > filters.clientWidth;
    const scrollLeft = filters.scrollLeft;
    const maxScroll = filters.scrollWidth - filters.clientWidth;

    const canScrollLeft = scrollLeft > 1;
    const canScrollRight = scrollLeft < maxScroll - 1;

    fadeLeft.classList.toggle('visible', canScrollLeft);
    fadeRight.classList.toggle('visible', canScrollRight && hasOverflow);
}

function updateFooterScroll(footer) {
    const hasOverflow = footer.scrollWidth > footer.clientWidth;
    const scrollLeft = footer.scrollLeft;
    const maxScroll = footer.scrollWidth - footer.clientWidth;

    const canScrollLeft = scrollLeft > 1;
    const canScrollRight = scrollLeft < maxScroll - 1;

    footer.classList.remove('no-scroll-left', 'no-scroll-right', 'no-scroll');

    if (!hasOverflow) {
        footer.classList.add('no-scroll');
    } else if (!canScrollLeft) {
        footer.classList.add('no-scroll-left');
    } else if (!canScrollRight) {
        footer.classList.add('no-scroll-right');
    }
}

function updateAllScrollIndicators() {
    updateScrollIndicators();
    const footers = document.querySelectorAll('.archive-footer');
    for (let i = 0; i < footers.length; i++) {
        updateFooterScroll(footers[i]);
    }
}

let currentArchives = '';

function getDisplayedArchives() {
    const cards = document.querySelectorAll('.archive-card');
    const names = [];
    for (let i = 0; i < cards.length; i++) {
        const href = cards[i].getAttribute('href');
        const match = href.match(/\/viewer\/([^\/]+)\//);
        if (match) {
            names.push(match[1]);
        }
    }
    return names.sort().join(',');
}

let connectionLostToast = null;

function checkUpdates() {
    fetch('/api/status')
        .then(res => {
            if (connectionLostToast) {
                hideToast(connectionLostToast);
                connectionLostToast = null;
            }
            return res.json();
        })
        .then(data => {
            const newArchives = data.archives.sort().join(',');
            if (currentArchives !== newArchives) {
                location.reload();
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
    const message = window.i18n && window.i18n.home && window.i18n.home.connection_lost ? window.i18n.home.connection_lost : 'Connection lost';
    connectionLostToast = showToast(message, 'error', 0);
}

function loadFilters() {
    const savedLanguage = localStorage.getItem('zimserver_filter_language');
    const savedCategory = localStorage.getItem('zimserver_filter_category');
    const savedSearch = localStorage.getItem('zimserver_filter_search');

    if (savedLanguage) {
        document.getElementById('languageFilter').value = savedLanguage;
    }
    if (savedCategory) {
        document.getElementById('categoryFilter').value = savedCategory;
    }
    if (savedSearch) {
        document.getElementById('searchBox').value = savedSearch;
    }
}

function setupSelectArrows() {
    const selects = document.querySelectorAll('select');
    
    document.addEventListener('click', (e) => {
        for (let i = 0; i < selects.length; i++) {
            const wrapper = selects[i].parentElement;
            if (wrapper.classList.contains('select-wrapper')) {
                if (!wrapper.contains(e.target)) {
                    wrapper.classList.remove('active');
                }
            }
        }
    });

    for (let i = 0; i < selects.length; i++) {
        const select = selects[i];
        const wrapper = select.parentElement;
        if (wrapper.classList.contains('select-wrapper')) {
            let justFocused = false;
            let justChanged = false;

            select.addEventListener('focus', () => {
                if (justChanged) return;
                wrapper.classList.add('active');
                justFocused = true;
                setTimeout(() => justFocused = false, 200);
            });

            select.addEventListener('blur', () => {
                wrapper.classList.remove('active');
            });

            select.addEventListener('click', () => {
                if (justChanged) {
                    justChanged = false;
                    return;
                }
                if (justFocused) {
                    wrapper.classList.add('active');
                    justFocused = false;
                } else {
                    wrapper.classList.toggle('active');
                }
            });
            
            select.addEventListener('change', () => {
                wrapper.classList.remove('active');
                select.blur();
                justChanged = true;
                setTimeout(() => justChanged = false, 200);
            });

            select.addEventListener('input', () => {
                wrapper.classList.remove('active');
                select.blur();
                justChanged = true;
                setTimeout(() => justChanged = false, 200);
            });
        }
    }
}

function setupSettingsModal() {
    const modal = document.getElementById('settingsModal');
    const btn = document.getElementById('settingsBtn');
    const closeBtn = document.getElementById('closeSettingsModal');
    const cancelBtn = document.getElementById('cancelSettings');
    const saveBtn = document.getElementById('saveSettings');
    const languageSelect = document.getElementById('languageSelect');
    const themeSelect = document.getElementById('themeSelect');
    const forceDarkModeCheckbox = document.getElementById('forceDarkModeCheckbox');

    const savedLang = localStorage.getItem('zimserver_language') || 'auto';
    const savedTheme = localStorage.getItem('zimserver_theme') || 'auto';
    const savedForceDarkMode = localStorage.getItem('zimserver_force_dark_mode') === 'true';
    
    languageSelect.value = savedLang;
    themeSelect.value = savedTheme;
    forceDarkModeCheckbox.checked = savedForceDarkMode;

    function updateForceDarkModeState() {
        const theme = themeSelect.value;
        const isSystemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        
        if (theme === 'light' || (theme === 'auto' && !isSystemDark)) {
            forceDarkModeCheckbox.disabled = true;
            forceDarkModeCheckbox.parentElement.parentElement.style.opacity = '0.5';
            forceDarkModeCheckbox.parentElement.parentElement.style.cursor = 'not-allowed';
        } else {
            forceDarkModeCheckbox.disabled = false;
            forceDarkModeCheckbox.parentElement.parentElement.style.opacity = '1';
            forceDarkModeCheckbox.parentElement.parentElement.style.cursor = 'pointer';
        }
    }

    themeSelect.addEventListener('change', updateForceDarkModeState);
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', updateForceDarkModeState);

    function openModal() {
        modal.classList.add('active');
        languageSelect.value = localStorage.getItem('zimserver_language') || 'auto';
        themeSelect.value = localStorage.getItem('zimserver_theme') || 'auto';
        forceDarkModeCheckbox.checked = localStorage.getItem('zimserver_force_dark_mode') === 'true';
        updateForceDarkModeState();
    }

    function closeModal() {
        modal.classList.remove('active');
    }

    function saveSettings() {
        const newLang = languageSelect.value;
        const oldLang = localStorage.getItem('zimserver_language') || 'auto';
        
        const newTheme = themeSelect.value;
        const oldTheme = localStorage.getItem('zimserver_theme') || 'auto';

        const newForceDarkMode = forceDarkModeCheckbox.checked;
        const oldForceDarkMode = localStorage.getItem('zimserver_force_dark_mode') === 'true';

        if (newLang === oldLang && newTheme === oldTheme && newForceDarkMode === oldForceDarkMode) {
            closeModal();
            return;
        }

        localStorage.setItem('zimserver_language', newLang);
        localStorage.setItem('zimserver_theme', newTheme);
        localStorage.setItem('zimserver_force_dark_mode', newForceDarkMode);
        
        document.cookie = `zimserver_language=${newLang}; path=/; max-age=31536000`; // 1 year
        
        applyTheme(newTheme);
        
        let needsReload = false;
        if (newLang !== oldLang) {
            if (newLang === 'auto') {
                const browserLang = navigator.language || navigator.userLanguage || '';
                const currentPrimary = (window.currentServerLang || '').split('-')[0].toLowerCase();
                const browserPrimary = browserLang.split('-')[0].toLowerCase();
                needsReload = currentPrimary !== browserPrimary;
            } else {
                needsReload = window.currentServerLang !== newLang;
            }
        }

        if (needsReload) {
            localStorage.setItem('zimserver_show_settings_saved', 'true');
            location.reload();
        } else {
            closeModal();
            showToast(window.i18n && window.i18n.home && window.i18n.home.settings_saved ? window.i18n.home.settings_saved : 'Settings saved', 'success');
        }
    }

    btn.addEventListener('click', openModal);
    closeBtn.addEventListener('click', closeModal);
    cancelBtn.addEventListener('click', closeModal);
    saveBtn.addEventListener('click', saveSettings);

    window.addEventListener('click', (e) => {
        if (e.target === modal) {
            closeModal();
        }
    });
}

function applyTheme(theme) {
    if (theme === 'auto') {
        document.documentElement.removeAttribute('data-theme');
    } else {
        document.documentElement.setAttribute('data-theme', theme);
    }
}

function setupImageLoading() {
    const images = document.querySelectorAll('.archive-icon img');
    for (let i = 0; i < images.length; i++) {
        const img = images[i];
        const container = img.parentElement;
        
        if (img.complete) {
            img.classList.add('loaded');
        } else {
            container.classList.add('loading');
            img.addEventListener('load', () => {
                img.classList.add('loaded');
                container.classList.remove('loading');
            }, { once: true });
            img.addEventListener('error', () => {
                container.classList.remove('loading');
            }, { once: true });
        }
    }
}

function showToast(message, type = 'info', duration = 3000) {
    const container = document.getElementById('toastContainer');
    if (!container) return null;

    const toast = document.createElement('div');
    toast.className = 'toast ' + type;
    
    let iconSvg = '';
    if (type === 'error') {
        iconSvg = `<svg class="toast-icon" xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 16 16">
          <path d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5m.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2"/>
        </svg>`;
    } else {
        iconSvg = `<svg class="toast-icon" xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 16 16">
          <path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14m0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16"/>
          <path d="m10.97 4.97-.02.022-3.473 4.425-2.093-2.094a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-1.071-1.05"/>
        </svg>`;
    }

    toast.innerHTML = `${iconSvg}<span>${message}</span>`;
    container.appendChild(toast);
    
    void toast.offsetWidth;
    toast.classList.add('show');
    
    if (duration > 0) {
        setTimeout(() => hideToast(toast), duration);
    }
    
    return toast;
}

function hideToast(toast) {
    if (!toast) return;
    toast.classList.remove('show');
    toast.classList.add('fade-out');
    setTimeout(() => {
        toast.remove();
    }, 200);
}

function checkPendingToast() {
    if (localStorage.getItem('zimserver_show_settings_saved') === 'true') {
        localStorage.removeItem('zimserver_show_settings_saved');
        showToast(window.i18n && window.i18n.home && window.i18n.home.settings_saved ? window.i18n.home.settings_saved : 'Settings saved', 'success');
    }
}

let currentTooltip = null;

function createTooltip(text) {
    const tooltip = document.createElement('div');
    tooltip.className = 'tooltip';
    
    const tooltipDescription = document.createElement('div');
    tooltipDescription.className = 'tooltip-description';
    tooltipDescription.textContent = text;
    
    tooltip.appendChild(tooltipDescription);
    
    return tooltip;
}

function showTooltip(element, text) {
    hideTooltip();
    
    currentTooltip = createTooltip(text);
    document.body.appendChild(currentTooltip);
    
    const rect = element.getBoundingClientRect();
    const tooltipRect = currentTooltip.getBoundingClientRect();
    
    let left = rect.left + (rect.width / 2) - (tooltipRect.width / 2);
    let top = rect.bottom + 10;
    let position = 'bottom';
    
    if (left < 10) left = 10;
    if (left + tooltipRect.width > window.innerWidth - 10) {
        left = window.innerWidth - tooltipRect.width - 10;
    }
    
    if (top + tooltipRect.height > window.innerHeight - 10) {
        top = rect.top - tooltipRect.height - 10;
        position = 'top';
    }
    
    currentTooltip.className = `tooltip ${position}`;
    
    currentTooltip.style.left = `${left}px`;
    currentTooltip.style.top = `${top}px`;
    
    const arrowOffset = Math.max(16, Math.min(tooltipRect.width - 16, rect.left + (rect.width / 2) - left));
    currentTooltip.style.setProperty('--arrow-offset', `${arrowOffset}px`);
    
    requestAnimationFrame(() => {
        if (currentTooltip) {
            currentTooltip.classList.add('show');
        }
    });
}

function hideTooltip() {
    if (currentTooltip) {
        const tooltip = currentTooltip;
        currentTooltip = null;
        tooltip.classList.remove('show');
        setTimeout(() => {
            tooltip.remove();
        }, 150);
    }
}

function attachTooltipEvents(element, text) {
    let hoverTimeout;
    
    element.addEventListener('mouseenter', () => {
        clearTimeout(hoverTimeout);
        hoverTimeout = setTimeout(() => {
            showTooltip(element, text);
        }, 200);
    });
    
    element.addEventListener('mouseleave', () => {
        clearTimeout(hoverTimeout);
        hideTooltip();
    });
    
    element.addEventListener('click', () => {
        hideTooltip();
    });
}

document.addEventListener('DOMContentLoaded', function() {
    loadFilters();
    filterArchives();
    setupSelectArrows();
    setupSettingsModal();
    setupImageLoading();
    checkPendingToast();

    const savedTheme = localStorage.getItem('zimserver_theme') || 'auto';
    applyTheme(savedTheme);

    const filters = document.querySelector('.filters');

    if (filters) {
        let scrollTimeout;
        filters.addEventListener('scroll', () => {
            clearTimeout(scrollTimeout);
            scrollTimeout = setTimeout(updateScrollIndicators, 50);
        }, { passive: true });
    }

    const footers = document.querySelectorAll('.archive-footer');
    for (let i = 0; i < footers.length; i++) {
        const footer = footers[i];
        footer.addEventListener('scroll', () => updateFooterScroll(footer), { passive: true });
        updateFooterScroll(footer);
    }

    let resizeTimeout;
    window.addEventListener('resize', () => {
        clearTimeout(resizeTimeout);
        resizeTimeout = setTimeout(updateAllScrollIndicators, 100);
    });

    currentArchives = getDisplayedArchives();
    setInterval(checkUpdates, 5000);

    const updateBtn = document.getElementById('updateBtn');
    if (updateBtn) {
        const tooltipText = window.i18n && window.i18n.home && window.i18n.home.check_for_updates 
            ? window.i18n.home.check_for_updates 
            : 'Check for updates';
        attachTooltipEvents(updateBtn, tooltipText);
    }
});
