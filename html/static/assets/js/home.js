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

function filterArchives() {
    const language = document.getElementById('languageFilter').value.toLowerCase();
    const category = document.getElementById('categoryFilter').value.toLowerCase();
    const search = document.getElementById('searchBox').value.toLowerCase();
    const cards = document.querySelectorAll('.archive-card');
    const clearBtn = document.getElementById('clearSearch');
    const noResults = document.getElementById('noResults');

    localStorage.setItem('zimserver_filter_language', document.getElementById('languageFilter').value);
    localStorage.setItem('zimserver_filter_category', document.getElementById('categoryFilter').value);
    localStorage.setItem('zimserver_filter_search', document.getElementById('searchBox').value);

    if (search) {
        clearBtn.classList.add('visible');
    } else {
        clearBtn.classList.remove('visible');
    }

    let visibleCount = 0;

    cards.forEach(card => {
        const cardLang = card.dataset.language.toLowerCase();
        const cardCategory = card.dataset.category.toLowerCase();
        const cardTags = card.dataset.tags.toLowerCase();
        const cardTitle = card.dataset.title.toLowerCase();
        const cardDesc = card.dataset.description.toLowerCase();

        const matchesLanguage = !language || cardLang === language || cardLang === 'mul';
        const matchesCategory = !category || cardCategory === category;
        const matchesSearch = !search || cardTitle.includes(search) || cardDesc.includes(search);

        if (matchesLanguage && matchesCategory && matchesSearch) {
            card.style.display = 'flex';
            void card.offsetWidth;
            card.classList.remove('fade-out');
            visibleCount++;
        } else {
            card.classList.add('fade-out');
            setTimeout(() => {
                if (card.classList.contains('fade-out')) {
                    card.style.display = 'none';
                }
            }, 300);
        }
    });

    if (noResultsTimeout) {
        clearTimeout(noResultsTimeout);
    }

    if (visibleCount === 0) {
        noResultsTimeout = setTimeout(() => {
            if (noResults) noResults.style.display = '';
        }, 300);
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
    
    setTimeout(updateAllScrollIndicators, 100);
}

function updateScrollIndicators() {
    const filters = document.querySelector('.filters');
    const fadeLeft = document.querySelector('.scroll-fade-left');
    const fadeRight = document.querySelector('.scroll-fade-right');

    const hasOverflow = filters.scrollWidth > filters.clientWidth;
    const scrollLeft = filters.scrollLeft;
    const maxScroll = filters.scrollWidth - filters.clientWidth;

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
    document.querySelectorAll('.archive-footer').forEach(updateFooterScroll);
}

let currentArchives = '';

function getDisplayedArchives() {
    const cards = document.querySelectorAll('.archive-card');
    const names = [];
    cards.forEach(card => {
        const href = card.getAttribute('href');
        const match = href.match(/\/viewer\/([^\/]+)\//);
        if (match) {
            names.push(match[1]);
        }
    });
    return names.sort().join(',');
}

function checkUpdates() {
    fetch('/api/status')
        .then(res => res.json())
        .then(data => {
            const newArchives = data.archives.sort().join(',');
            if (currentArchives !== newArchives) {
                location.reload();
            }
        })
        .catch(console.error);
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
        selects.forEach(select => {
            const wrapper = select.parentElement;
            if (wrapper.classList.contains('select-wrapper')) {
                if (!wrapper.contains(e.target)) {
                    wrapper.classList.remove('active');
                }
            }
        });
    });

    selects.forEach(select => {
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
    });
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
            forceDarkModeCheckbox.parentElement.style.opacity = '0.5';
            forceDarkModeCheckbox.parentElement.style.cursor = 'not-allowed';
        } else {
            forceDarkModeCheckbox.disabled = false;
            forceDarkModeCheckbox.parentElement.style.opacity = '1';
            forceDarkModeCheckbox.parentElement.style.cursor = 'pointer';
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

        localStorage.setItem('zimserver_language', newLang);
        localStorage.setItem('zimserver_theme', newTheme);
        localStorage.setItem('zimserver_force_dark_mode', newForceDarkMode);
        
        document.cookie = `zimserver_language=${newLang}; path=/; max-age=31536000`; // 1 year
        
        applyTheme(newTheme);
        
        if (newLang !== oldLang) {
            location.reload();
        } else {
            closeModal();
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
    images.forEach(img => {
        const container = img.parentElement;
        container.classList.add('loading');
        
        if (img.complete) {
            img.classList.add('loaded');
            container.classList.remove('loading');
        } else {
            img.addEventListener('load', () => {
                img.classList.add('loaded');
                container.classList.remove('loading');
            });
            img.addEventListener('error', () => {
                container.classList.remove('loading');
            });
        }
    });
}

document.addEventListener('DOMContentLoaded', function() {
    loadFilters();
    filterArchives();
    setupSelectArrows();
    setupSettingsModal();
    setupImageLoading();
    
    const savedTheme = localStorage.getItem('zimserver_theme') || 'auto';
    applyTheme(savedTheme);

    const filters = document.querySelector('.filters');

    setTimeout(updateAllScrollIndicators, 100);

    filters.addEventListener('scroll', updateScrollIndicators);
    
    document.querySelectorAll('.archive-footer').forEach(footer => {
        footer.addEventListener('scroll', () => updateFooterScroll(footer));
        updateFooterScroll(footer);
    });

    window.addEventListener('resize', () => {
        setTimeout(updateAllScrollIndicators, 100);
    });

    currentArchives = getDisplayedArchives();
    setInterval(checkUpdates, 5000);
});
