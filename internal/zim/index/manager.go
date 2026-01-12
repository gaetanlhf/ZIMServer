package index

import (
	"fmt"
	"math/rand"
	"time"

	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

type Manager struct {
	reader  *zimreader.ZIMReader
	titleV0 *Index
	titleV1 *Index
	hasV0   bool
	hasV1   bool
	rng     *rand.Rand
}

func NewManager(reader *zimreader.ZIMReader) (*Manager, error) {
	mgr := &Manager{
		reader: reader,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	titleV0, err := NewIndex(reader, IndexTypeTitleV0)
	if err == nil {
		mgr.titleV0 = titleV0
		mgr.hasV0 = true
	}

	titleV1, err := NewIndex(reader, IndexTypeTitleV1)
	if err == nil {
		mgr.titleV1 = titleV1
		mgr.hasV1 = true
	}

	if !mgr.hasV0 && !mgr.hasV1 {
		return nil, fmt.Errorf("no title index available")
	}

	return mgr, nil
}

func (m *Manager) HasTitleV0() bool {
	return m.hasV0
}

func (m *Manager) HasTitleV1() bool {
	return m.hasV1
}

func (m *Manager) Search(query string, maxResults int) ([]SearchResult, error) {
	if m.hasV1 {
		return m.titleV1.Search(query, maxResults)
	}
	if m.hasV0 {
		return m.titleV0.Search(query, maxResults)
	}
	return nil, fmt.Errorf("no index available")
}

func (m *Manager) SearchByTitle(titlePrefix string, maxResults int) ([]SearchResult, error) {
	if m.hasV1 {
		return m.titleV1.SearchByTitle(titlePrefix, maxResults)
	}
	if m.hasV0 {
		return m.titleV0.SearchByTitle(titlePrefix, maxResults)
	}
	return nil, fmt.Errorf("no index available")
}

func (m *Manager) SearchArticles(query string, maxResults int) ([]SearchResult, error) {
	if !m.hasV1 {
		return nil, fmt.Errorf("article index (v1) not available")
	}
	return m.titleV1.Search(query, maxResults)
}

func (m *Manager) GetRandomArticle() (zimreader.DirectoryEntry, error) {
	maxAttempts := 100

	for attempt := 0; attempt < maxAttempts; attempt++ {
		var entry zimreader.DirectoryEntry
		var err error

		if m.hasV1 {
			if m.titleV1.Size() == 0 {
				return nil, fmt.Errorf("no articles in index")
			}
			randomPos := m.rng.Intn(m.titleV1.Size())
			entry, err = m.titleV1.GetEntry(randomPos)
		} else if m.hasV0 {
			if m.titleV0.Size() == 0 {
				return nil, fmt.Errorf("no articles in index")
			}
			randomPos := m.rng.Intn(m.titleV0.Size())
			entry, err = m.titleV0.GetEntry(randomPos)
		} else {
			return nil, fmt.Errorf("no index available")
		}

		if err != nil {
			continue
		}

		resolvedEntry, err := m.reader.ResolveRedirect(entry)
		if err != nil {
			continue
		}

		return resolvedEntry, nil
	}

	return nil, fmt.Errorf("could not find article after %d attempts", maxAttempts)
}

func (m *Manager) GetRandomArticles(count int) ([]zimreader.DirectoryEntry, error) {
	if !m.hasV1 {
		return nil, fmt.Errorf("article index (v1) not available")
	}

	size := m.titleV1.Size()
	if size == 0 {
		return nil, fmt.Errorf("no articles in index")
	}

	if count > size {
		count = size
	}

	entries := make([]zimreader.DirectoryEntry, 0, count)
	used := make(map[int]bool)
	seen := make(map[string]bool)
	maxAttempts := count * 10

	attempt := 0
	for len(entries) < count && attempt < maxAttempts {
		attempt++

		pos := m.rng.Intn(size)
		if used[pos] {
			continue
		}
		used[pos] = true

		entry, err := m.titleV1.GetEntry(pos)
		if err != nil {
			continue
		}

		resolvedEntry, err := m.reader.ResolveRedirect(entry)
		if err != nil {
			continue
		}

		key := string(resolvedEntry.GetNamespace()) + resolvedEntry.GetPath()
		if seen[key] {
			continue
		}
		seen[key] = true

		entries = append(entries, resolvedEntry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("could not find any articles")
	}

	return entries, nil
}
