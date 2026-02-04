function toggleModal() {
    document.getElementById('infoModal').classList.toggle('active');
}

function clearSearch() {
    document.getElementById('searchBox').value = '';
    filterArchives();
}

function filterArchives() {
    const language = document.getElementById('languageFilter').value.toLowerCase();
    const category = document.getElementById('categoryFilter').value.toLowerCase();
    const search = document.getElementById('searchBox').value.toLowerCase();
    const cards = document.querySelectorAll('.archive-card');
    const clearBtn = document.getElementById('clearSearch');

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
        const matchesCategory = !category || cardCategory.includes(category) || cardTags.includes(category);
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

    const plural = visibleCount === 1 ? 'archive' : 'archives';
    document.getElementById('archiveCount').textContent = visibleCount + ' ' + plural;
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

document.addEventListener('DOMContentLoaded', function() {
    document.getElementById('infoModal').addEventListener('click', function(e) {
        if (e.target === this) {
            toggleModal();
        }
    });

    const searchBox = document.getElementById('searchBox');
    if (searchBox.value) {
        filterArchives();
    }

    const filters = document.querySelector('.filters');

    setTimeout(updateScrollIndicators, 100);

    filters.addEventListener('scroll', updateScrollIndicators);
    window.addEventListener('resize', () => {
        setTimeout(updateScrollIndicators, 100);
    });

    currentArchives = getDisplayedArchives();
    setInterval(checkUpdates, 5000);
});
