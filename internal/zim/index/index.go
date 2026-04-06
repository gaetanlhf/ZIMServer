package index

import (
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"

	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

func NewIndex(reader *zimreader.ZIMReader, indexType IndexType) (*Index, error) {
	entry, err := reader.GetEntryByURL(zimreader.NamespaceIndex, string(indexType))
	if err != nil {
		return nil, fmt.Errorf("index not found: %w", err)
	}

	return &Index{
		reader:     reader,
		entry:      entry,
		entryCount: 0,
	}, nil
}

func (idx *Index) Size() int {
	if idx.entryCount > 0 {
		return idx.entryCount
	}

	_, size, err := idx.reader.GetContentReader(idx.entry)
	if err != nil {
		return 0
	}
	idx.entryCount = int(size) / 4
	return idx.entryCount
}

func (idx *Index) GetEntry(position int) (zimreader.DirectoryEntry, error) {
	readerAt, _, err := idx.reader.GetContentReader(idx.entry)
	if err != nil {
		return nil, err
	}

	offset := int64(position * 4)
	buf := make([]byte, 4)
	if _, err := readerAt.ReadAt(buf, offset); err != nil {
		return nil, fmt.Errorf("failed to read index entry at %d: %w", position, err)
	}

	entryIndex := binary.LittleEndian.Uint32(buf)
	return idx.reader.GetEntryByIndex(entryIndex)
}

func (idx *Index) Search(query string, maxResults int) ([]SearchResult, error) {
	return idx.SearchByTitle(query, maxResults)
}

func (idx *Index) SearchByTitle(titlePrefix string, maxResults int) ([]SearchResult, error) {
	titlePrefix = strings.ToLower(strings.TrimSpace(titlePrefix))
	if titlePrefix == "" {
		return nil, fmt.Errorf("empty title prefix")
	}

	readerAt, size, err := idx.reader.GetContentReader(idx.entry)
	if err != nil {
		return nil, err
	}

	idx.entryCount = int(size) / 4

	capacity := maxResults
	if capacity <= 0 {
		capacity = 100
	}
	results := make([]SearchResult, 0, capacity)
	seen := make(map[string]bool)

	start := idx.binarySearchTitle(readerAt, idx.entryCount, titlePrefix)

	chunkSize := 1024
	buf := make([]byte, chunkSize*4)

	for i := start; i < idx.entryCount; {
		if maxResults > 0 && len(results) >= maxResults {
			break
		}

		remaining := idx.entryCount - i
		toRead := chunkSize
		if remaining < toRead {
			toRead = remaining
		}

		readBytes := toRead * 4
		if _, err := readerAt.ReadAt(buf[:readBytes], int64(i*4)); err != nil {
			break
		}

		for j := 0; j < toRead; j++ {
			if maxResults > 0 && len(results) >= maxResults {
				break
			}

			entryIndex := binary.LittleEndian.Uint32(buf[j*4 : (j+1)*4])
			entry, err := idx.reader.GetEntryByIndex(entryIndex)
			if err != nil {
				continue
			}

			title := strings.ToLower(entry.GetTitle())
			if !strings.HasPrefix(title, titlePrefix) {
				if title > titlePrefix {
					sortResultsByScore(results)
					return results, nil
				}
				continue
			}

			resolvedEntry, err := idx.reader.ResolveRedirect(entry)
			if err != nil {
				continue
			}

			key := string(resolvedEntry.GetNamespace()) + resolvedEntry.GetPath()
			if !seen[key] {
				seen[key] = true

				results = append(results, SearchResult{
					Index: uint32(i + j),
					Entry: resolvedEntry,
					Score: 1.0,
				})
			}
		}
		i += toRead
	}

	sortResultsByScore(results)

	return results, nil
}

func (idx *Index) binarySearchTitle(r io.ReaderAt, count int, prefix string) int {
	left, right := 0, count
	buf := make([]byte, 4)

	for left < right {
		mid := (left + right) / 2

		if _, err := r.ReadAt(buf, int64(mid*4)); err != nil {
			left = mid + 1
			continue
		}

		entryIndex := binary.LittleEndian.Uint32(buf)
		entry, err := idx.reader.GetEntryByIndex(entryIndex)
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

func sortResultsByScore(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
}
