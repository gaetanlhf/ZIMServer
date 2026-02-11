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

    // Save filters to localStorage
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

    const plural = visibleCount === 1 ? 'archive' : 'archives';
    document.getElementById('archiveCount').textContent = visibleCount + ' ' + plural;
    
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

document.addEventListener('DOMContentLoaded', function() {
    loadFilters();
    filterArchives();

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
