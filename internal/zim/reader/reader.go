package reader

import (
	"encoding/binary"
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

	zr, err := NewReaderFromReaderAt(file)
	if err != nil {
		file.Close()
		return nil, err
	}

	return zr, nil
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

	return zr, nil
}

func (zr *ZIMReader) Close() error {
	if c, ok := zr.file.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (zr *ZIMReader) GetHeader() *Header {
	return zr.header
}

func (zr *ZIMReader) GetMimeTypes() []string {
	return zr.mimeTypes
}

func (zr *ZIMReader) GetEntryByPath(path string) (DirectoryEntry, error) {
	count := int(zr.header.EntryCount)
	idx := sort.Search(count, func(i int) bool {
		ptr, err := zr.readPathPointer(i)
		if err != nil {
			return false
		}
		entry, err := readDirectoryEntry(zr.file, ptr)
		if err != nil {
			return false
		}
		fullPath := string(entry.GetNamespace()) + entry.GetPath()
		return fullPath >= path
	})

	if idx >= count {
		return nil, fmt.Errorf("entry not found: %s", path)
	}

	ptr, err := zr.readPathPointer(idx)
	if err != nil {
		return nil, err
	}

	entry, err := readDirectoryEntry(zr.file, ptr)
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
	if index >= zr.header.EntryCount {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	ptr, err := zr.readPathPointer(int(index))
	if err != nil {
		return nil, err
	}

	return readDirectoryEntry(zr.file, ptr)
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

	redirectEntry, ok := entry.(*RedirectEntry)
	if !ok {
		return nil, fmt.Errorf("entry is marked as redirect but has unexpected type")
	}

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

	contentEntry, ok := resolvedEntry.(*ContentEntry)
	if !ok {
		return nil, fmt.Errorf("resolved entry has unexpected type")
	}
	cluster, err := zr.getCluster(contentEntry.ClusterNumber)
	if err != nil {
		return nil, err
	}

	return cluster.ReadBlob(contentEntry.BlobNumber)
}

func (zr *ZIMReader) GetContentReader(entry DirectoryEntry) (io.ReaderAt, int64, error) {
	resolvedEntry, err := zr.ResolveRedirect(entry)
	if err != nil {
		return nil, 0, err
	}

	contentEntry, ok := resolvedEntry.(*ContentEntry)
	if !ok {
		return nil, 0, fmt.Errorf("resolved entry has unexpected type")
	}
	cluster, err := zr.getCluster(contentEntry.ClusterNumber)
	if err != nil {
		return nil, 0, err
	}

	return cluster.BlobReader(contentEntry.BlobNumber)
}

func (zr *ZIMReader) GetMimeType(entry DirectoryEntry) (string, error) {
	resolvedEntry, err := zr.ResolveRedirect(entry)
	if err != nil {
		return "", err
	}

	contentEntry, ok := resolvedEntry.(*ContentEntry)
	if !ok {
		return "", fmt.Errorf("resolved entry has unexpected type")
	}
	if contentEntry.MimeType >= uint16(len(zr.mimeTypes)) {
		return "", fmt.Errorf("invalid mime type index: %d", contentEntry.MimeType)
	}

	return zr.mimeTypes[contentEntry.MimeType], nil
}

func (zr *ZIMReader) getCluster(index uint32) (*Cluster, error) {
	if index >= zr.header.ClusterCount {
		return nil, fmt.Errorf("cluster index out of range: %d", index)
	}

	offset, err := zr.readClusterPointer(int(index))
	if err != nil {
		return nil, err
	}

	var size uint64
	if index+1 < zr.header.ClusterCount {
		nextOffset, err := zr.readClusterPointer(int(index + 1))
		if err != nil {
			return nil, err
		}
		size = nextOffset - offset
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
	
	for i := 0; i < int(zr.header.EntryCount); i++ {
		ptr, err := zr.readPathPointer(i)
		if err != nil {
			continue
		}

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

func (zr *ZIMReader) readPathPointer(index int) (uint64, error) {
	offset := int64(zr.header.PathPtrPos) + int64(index*8)
	buf := make([]byte, 8)
	if _, err := zr.file.ReadAt(buf, offset); err != nil {
		return 0, fmt.Errorf("failed to read path pointer at index %d: %w", index, err)
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func (zr *ZIMReader) readClusterPointer(index int) (uint64, error) {
	offset := int64(zr.header.ClusterPtrPos) + int64(index*8)
	buf := make([]byte, 8)
	if _, err := zr.file.ReadAt(buf, offset); err != nil {
		return 0, fmt.Errorf("failed to read cluster pointer at index %d: %w", index, err)
	}
	return binary.LittleEndian.Uint64(buf), nil
}
