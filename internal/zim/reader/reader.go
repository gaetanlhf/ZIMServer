package reader

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func NewReader(filename string) (*ZIMReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return NewReaderFromReaderAt(file)
}

func NewReaderFromReaderAt(r io.ReaderAt) (*ZIMReader, error) {
	zr := &ZIMReader{file: r}

	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	zr.header = header

	mimeTypes, err := readMimeTypes(r, header.MimeListPos)
	if err != nil {
		return nil, err
	}
	zr.mimeTypes = mimeTypes

	pathPointers, err := readPathPointers(r, header.PathPtrPos, header.EntryCount)
	if err != nil {
		return nil, err
	}
	zr.pathPointers = pathPointers

	if header.TitlePtrPos != 0xffffffffffffffff {
		titlePointers, err := readTitlePointers(r, header.TitlePtrPos, header.EntryCount)
		if err != nil {
			return nil, err
		}
		zr.titlePointers = titlePointers
	}

	clusterPtrs, err := readClusterPointers(r, header.ClusterPtrPos, header.ClusterCount)
	if err != nil {
		return nil, err
	}
	zr.clusterPtrs = clusterPtrs

	return zr, nil
}

func (zr *ZIMReader) GetHeader() *Header {
	return zr.header
}

func (zr *ZIMReader) GetMimeTypes() []string {
	return zr.mimeTypes
}

func (zr *ZIMReader) GetEntryByPath(path string) (DirectoryEntry, error) {
	idx := sort.Search(len(zr.pathPointers), func(i int) bool {
		entry, err := readDirectoryEntry(zr.file, zr.pathPointers[i])
		if err != nil {
			return false
		}
		fullPath := string(entry.GetNamespace()) + entry.GetPath()
		return fullPath >= path
	})

	if idx >= len(zr.pathPointers) {
		return nil, fmt.Errorf("entry not found: %s", path)
	}

	entry, err := readDirectoryEntry(zr.file, zr.pathPointers[idx])
	if err != nil {
		return nil, err
	}

	fullPath := string(entry.GetNamespace()) + entry.GetPath()
	if fullPath != path {
		return nil, fmt.Errorf("entry not found: %s", path)
	}

	return entry, nil
}

func (zr *ZIMReader) GetEntryByIndex(index uint32) (DirectoryEntry, error) {
	if index >= uint32(len(zr.pathPointers)) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	return readDirectoryEntry(zr.file, zr.pathPointers[index])
}

func (zr *ZIMReader) GetEntryByURL(namespace byte, path string) (DirectoryEntry, error) {
	fullPath := string(namespace) + path
	return zr.GetEntryByPath(fullPath)
}

func (zr *ZIMReader) GetMainPage() (DirectoryEntry, error) {
	if zr.header.MainPage == 0xffffffff {
		return nil, fmt.Errorf("no main page defined")
	}
	return zr.GetEntryByIndex(zr.header.MainPage)
}

func (zr *ZIMReader) ResolveRedirect(entry DirectoryEntry) (DirectoryEntry, error) {
	return zr.resolveRedirectWithDepth(entry, 0, 10)
}

func (zr *ZIMReader) resolveRedirectWithDepth(entry DirectoryEntry, depth, maxDepth int) (DirectoryEntry, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("maximum redirect depth exceeded (%d redirects)", maxDepth)
	}

	if !entry.IsRedirect() {
		return entry, nil
	}

	redirectEntry := entry.(*RedirectEntry)

	if redirectEntry.RedirectIndex == 0xffffffff {
		return nil, fmt.Errorf("invalid redirect index: 0xffffffff")
	}

	targetEntry, err := zr.GetEntryByIndex(redirectEntry.RedirectIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve redirect: %w", err)
	}

	return zr.resolveRedirectWithDepth(targetEntry, depth+1, maxDepth)
}

func (zr *ZIMReader) GetContent(entry DirectoryEntry) ([]byte, error) {
	resolvedEntry, err := zr.ResolveRedirect(entry)
	if err != nil {
		return nil, err
	}

	contentEntry := resolvedEntry.(*ContentEntry)
	cluster, err := zr.getCluster(contentEntry.ClusterNumber)
	if err != nil {
		return nil, err
	}

	return cluster.ReadBlob(contentEntry.BlobNumber)
}

func (zr *ZIMReader) GetMimeType(entry DirectoryEntry) (string, error) {
	resolvedEntry, err := zr.ResolveRedirect(entry)
	if err != nil {
		return "", err
	}

	contentEntry := resolvedEntry.(*ContentEntry)
	if contentEntry.MimeType >= uint16(len(zr.mimeTypes)) {
		return "", fmt.Errorf("invalid mime type index: %d", contentEntry.MimeType)
	}

	return zr.mimeTypes[contentEntry.MimeType], nil
}

func (zr *ZIMReader) getCluster(index uint32) (*Cluster, error) {
	if index >= uint32(len(zr.clusterPtrs)) {
		return nil, fmt.Errorf("cluster index out of range: %d", index)
	}

	offset := zr.clusterPtrs[index]
	var size uint64
	if index+1 < uint32(len(zr.clusterPtrs)) {
		size = zr.clusterPtrs[index+1] - offset
	} else {
		size = zr.header.ChecksumPos - offset
	}

	return &Cluster{
		reader: zr.file,
		offset: offset,
		size:   size,
	}, nil
}

func (zr *ZIMReader) ListEntriesByNamespace(namespace byte) ([]DirectoryEntry, error) {
	var entries []DirectoryEntry
	prefix := string(namespace)

	for _, ptr := range zr.pathPointers {
		entry, err := readDirectoryEntry(zr.file, ptr)
		if err != nil {
			continue
		}

		fullPath := string(entry.GetNamespace()) + entry.GetPath()
		if strings.HasPrefix(fullPath, prefix) {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (zr *ZIMReader) GetMetadata(key string) (string, error) {
	entry, err := zr.GetEntryByURL(NamespaceMetadata, key)
	if err != nil {
		return "", err
	}

	content, err := zr.GetContent(entry)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
