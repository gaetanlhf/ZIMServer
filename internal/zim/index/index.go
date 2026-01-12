package index

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"

	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

func NewIndex(reader *zimreader.ZIMReader, indexType IndexType) (*Index, error) {
	entry, err := reader.GetEntryByURL(zimreader.NamespaceIndex, string(indexType))
	if err != nil {
		return nil, fmt.Errorf("index not found: %w", err)
	}

	content, err := reader.GetContent(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to read index content: %w", err)
	}

	if len(content)%4 != 0 {
		return nil, fmt.Errorf("invalid index size: %d bytes", len(content))
	}

	entryCount := len(content) / 4
	entries := make([]uint32, entryCount)

	for i := 0; i < entryCount; i++ {
		offset := i * 4
		entries[i] = binary.LittleEndian.Uint32(content[offset : offset+4])
	}

	return &Index{
		reader:  reader,
		entries: entries,
	}, nil
}

func (idx *Index) Size() int {
	return len(idx.entries)
}

func (idx *Index) GetEntry(position int) (zimreader.DirectoryEntry, error) {
	if position < 0 || position >= len(idx.entries) {
		return nil, fmt.Errorf("position out of bounds: %d", position)
	}

	entryIndex := idx.entries[position]
	return idx.reader.GetEntryByIndex(entryIndex)
}

func (idx *Index) Search(query string, maxResults int) ([]SearchResult, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	results := make([]SearchResult, 0, maxResults*2)
	seen := make(map[string]bool)

	for i, entryIndex := range idx.entries {
		entry, err := idx.reader.GetEntryByIndex(entryIndex)
		if err != nil {
			continue
		}

		resolvedEntry, err := idx.reader.ResolveRedirect(entry)
		if err != nil {
			continue
		}

		title := strings.ToLower(entry.GetTitle())
		path := strings.ToLower(entry.GetPath())

		score := calculateScore(query, title, path)
		if score > 0 {
			key := string(resolvedEntry.GetNamespace()) + resolvedEntry.GetPath()
			if !seen[key] {
				seen[key] = true
				results = append(results, SearchResult{
					Index: uint32(i),
					Entry: resolvedEntry,
					Score: score,
				})
			}
		}
	}

	sortResultsByScore(results)

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

func (idx *Index) SearchByTitle(titlePrefix string, maxResults int) ([]SearchResult, error) {
	titlePrefix = strings.ToLower(strings.TrimSpace(titlePrefix))
	if titlePrefix == "" {
		return nil, fmt.Errorf("empty title prefix")
	}

	results := make([]SearchResult, 0, maxResults)
	seen := make(map[string]bool)

	start := idx.binarySearchTitle(titlePrefix)

	for i := start; i < len(idx.entries) && len(results) < maxResults; i++ {
		entry, err := idx.reader.GetEntryByIndex(idx.entries[i])
		if err != nil {
			continue
		}

		title := strings.ToLower(entry.GetTitle())
		if !strings.HasPrefix(title, titlePrefix) {
			break
		}

		resolvedEntry, err := idx.reader.ResolveRedirect(entry)
		if err != nil {
			continue
		}

		key := string(resolvedEntry.GetNamespace()) + resolvedEntry.GetPath()
		if !seen[key] {
			seen[key] = true

			results = append(results, SearchResult{
				Index: uint32(i),
				Entry: resolvedEntry,
				Score: 1.0,
			})
		}
	}

	sortResultsByScore(results)

	return results, nil
}

func (idx *Index) binarySearchTitle(prefix string) int {
	left, right := 0, len(idx.entries)

	for left < right {
		mid := (left + right) / 2
		entry, err := idx.reader.GetEntryByIndex(idx.entries[mid])
		if err != nil {
			left = mid + 1
			continue
		}

		title := strings.ToLower(entry.GetTitle())
		if title < prefix {
			left = mid + 1
		} else {
			right = mid
		}
	}

	return left
}

func calculateScore(query, title, path string) float64 {
	score := 0.0

	if title == query {
		score += 10.0
	}

	if strings.HasPrefix(title, query) {
		score += 5.0
	}

	if strings.Contains(title, query) {
		score += 2.0
	}

	if strings.Contains(path, query) {
		score += 0.5
	}

	queryWords := strings.Fields(query)
	titleWords := strings.Fields(title)

	matchCount := 0
	for _, qw := range queryWords {
		for _, tw := range titleWords {
			if strings.Contains(tw, qw) {
				matchCount++
				break
			}
		}
	}

	if len(queryWords) > 0 {
		score += float64(matchCount) / float64(len(queryWords))
	}

	return score
}

func sortResultsByScore(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}
