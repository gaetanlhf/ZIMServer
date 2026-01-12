package index

import (
	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

type IndexType string

const (
	IndexTypeTitleV0 IndexType = "listing/titleOrdered/v0"
	IndexTypeTitleV1 IndexType = "listing/titleOrdered/v1"
)

type Index struct {
	reader  *zimreader.ZIMReader
	entries []uint32
}

type SearchResult struct {
	Index     uint32
	Entry     zimreader.DirectoryEntry
	Score     float64
	Snippet   string
	WordCount int
	Source    string
}
