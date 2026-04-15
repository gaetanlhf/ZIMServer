package reader

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"golang.org/x/exp/mmap"
)

var contentNamespaces = []byte{NamespaceContent, NamespaceMetadata, NamespaceWellKnown, NamespaceIndex}

func NewReader(filename string) (*ZIMReader, error) {
	if mapped, err := mmap.Open(filename); err == nil {
		zr, zerr := NewReaderFromReaderAt(mapped)
		if zerr != nil {
			mapped.Close()
			return nil, zerr
		}
		return zr, nil
	}
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
	zr := &ZIMReader{
		file:        r,
		cache:       newClusterCache(defaultClusterCacheEntries),
		direntCache: newDirentCache(defaultDirentCacheEntries),
	}
	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	zr.header = header

	mimeTypes, err := readMimeTypes(r, header.MimeListPos, zr.mimeListEnd())
	if err != nil {
		return nil, err
	}
	zr.mimeTypes = mimeTypes

	if header.ChecksumPos != 0 {
		size := readerSize(r)
		if size > 0 && header.ChecksumPos+16 > size {
			return nil, fmt.Errorf("invalid header: ChecksumPos %d + 16 exceeds file size %d", header.ChecksumPos, size)
		}
	}
	return zr, nil
}

func readerSize(r io.ReaderAt) uint64 {
	if sz, ok := r.(interface{ Len() int }); ok {
		return uint64(sz.Len())
	}
	if sz, ok := r.(interface{ Stat() (os.FileInfo, error) }); ok {
		if info, err := sz.Stat(); err == nil {
			return uint64(info.Size())
		}
	}
	return 0
}

func (zr *ZIMReader) mimeListEnd() uint64 {
	end := zr.header.PathPtrPos
	if zr.header.TitlePtrPos != ^uint64(0) && zr.header.TitlePtrPos < end {
		end = zr.header.TitlePtrPos
	}
	if zr.header.ClusterPtrPos < end {
		end = zr.header.ClusterPtrPos
	}
	return end
}

func (zr *ZIMReader) clusterRegionEnd() uint64 {
	if zr.header.ChecksumPos != 0 {
		return zr.header.ChecksumPos
	}
	return readerSize(zr.file)
}

func (zr *ZIMReader) Close() error {
	if c, ok := zr.file.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (zr *ZIMReader) GetHeader() *Header        { return zr.header }
func (zr *ZIMReader) GetMimeTypes() []string    { return zr.mimeTypes }
func (zr *ZIMReader) ContentNamespaces() []byte { return contentNamespaces }

func (zr *ZIMReader) GetEntryByPath(fullPath string) (DirectoryEntry, error) {
	count := int(zr.header.EntryCount)
	idx, err := zr.searchPathLowerBound(count, fullPath)
	if err != nil {
		return nil, err
	}
	if idx >= count {
		return nil, fmt.Errorf("entry not found: %s", fullPath)
	}
	entry, err := zr.entryAtIndex(idx)
	if err != nil {
		return nil, err
	}
	if entryFullPath(entry) != fullPath {
		return nil, fmt.Errorf("entry not found: %s", fullPath)
	}
	return entry, nil
}

func (zr *ZIMReader) GetEntryByURL(namespace byte, path string) (DirectoryEntry, error) {
	return zr.GetEntryByPath(string(namespace) + path)
}

func (zr *ZIMReader) GetEntryByIndex(index uint32) (DirectoryEntry, error) {
	if index >= zr.header.EntryCount {
		return nil, fmt.Errorf("index out of range: %d", index)
	}
	return zr.entryAtIndex(int(index))
}

func (zr *ZIMReader) FindEntry(path string) (DirectoryEntry, error) {
	for _, ns := range contentNamespaces {
		if entry, err := zr.GetEntryByURL(ns, path); err == nil {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("entry not found: %s", path)
}

func (zr *ZIMReader) entryAtIndex(index int) (DirectoryEntry, error) {
	if entry, ok := zr.direntCache.Get(index); ok {
		return entry, nil
	}
	ptr, err := zr.readPathPointer(index)
	if err != nil {
		return nil, err
	}
	entry, err := readDirectoryEntry(zr.file, ptr)
	if err != nil {
		return nil, err
	}
	zr.direntCache.Put(index, entry)
	return entry, nil
}

func entryFullPath(entry DirectoryEntry) string {
	return string(entry.GetNamespace()) + entry.GetPath()
}

func (zr *ZIMReader) searchPathLowerBound(count int, fullPath string) (int, error) {
	zr.ensurePathGrid()
	left, right := 0, count
	if zr.pathGrid != nil {
		r := zr.pathGrid.lookup(fullPath)
		if r.begin <= r.end && r.end <= count {
			left, right = r.begin, r.end
		}
	}
	for left < right {
		mid := (left + right) / 2
		entry, err := zr.entryAtIndex(mid)
		if err != nil {
			return 0, err
		}
		if entryFullPath(entry) < fullPath {
			left = mid + 1
		} else {
			right = mid
		}
	}
	return left, nil
}

func (zr *ZIMReader) GetMainPage() (DirectoryEntry, error) {
	if !zr.header.HasMainPage() {
		return nil, fmt.Errorf("no main page defined")
	}
	return zr.GetEntryByIndex(zr.header.MainPage)
}

func (zr *ZIMReader) ResolveRedirect(entry DirectoryEntry) (DirectoryEntry, error) {
	for depth := 0; depth < 16; depth++ {
		if !entry.IsRedirect() {
			return entry, nil
		}
		re, ok := entry.(*RedirectEntry)
		if !ok {
			return nil, fmt.Errorf("redirect entry has unexpected type")
		}
		next, err := zr.GetEntryByIndex(re.RedirectIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve redirect: %w", err)
		}
		entry = next
	}
	return nil, fmt.Errorf("redirect chain too deep")
}

func (zr *ZIMReader) GetContent(entry DirectoryEntry) ([]byte, error) {
	resolved, err := zr.ResolveRedirect(entry)
	if err != nil {
		return nil, err
	}
	ce, ok := resolved.(*ContentEntry)
	if !ok {
		return nil, fmt.Errorf("entry %q has no content", resolved.GetPath())
	}
	cluster, err := zr.getCluster(ce.ClusterNumber)
	if err != nil {
		return nil, err
	}
	return cluster.ReadBlob(ce.BlobNumber)
}

func (zr *ZIMReader) GetContentReader(entry DirectoryEntry) (io.ReaderAt, int64, error) {
	resolved, err := zr.ResolveRedirect(entry)
	if err != nil {
		return nil, 0, err
	}
	ce, ok := resolved.(*ContentEntry)
	if !ok {
		return nil, 0, fmt.Errorf("entry %q has no content", resolved.GetPath())
	}
	cluster, err := zr.getCluster(ce.ClusterNumber)
	if err != nil {
		return nil, 0, err
	}
	return cluster.BlobReader(ce.BlobNumber)
}

func (zr *ZIMReader) GetMimeType(entry DirectoryEntry) (string, error) {
	resolved, err := zr.ResolveRedirect(entry)
	if err != nil {
		return "", err
	}
	ce, ok := resolved.(*ContentEntry)
	if !ok {
		return "application/octet-stream", nil
	}
	if ce.MimeType >= uint16(len(zr.mimeTypes)) {
		return "application/octet-stream", nil
	}
	return zr.mimeTypes[ce.MimeType], nil
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
		next, err := zr.readClusterPointer(int(index + 1))
		if err != nil {
			return nil, err
		}
		size = next - offset
	} else {
		end := zr.clusterRegionEnd()
		if end <= offset {
			return nil, fmt.Errorf("cannot determine size of last cluster")
		}
		size = end - offset
	}
	return &Cluster{
		reader: zr.file,
		offset: offset,
		size:   size,
		index:  index,
		cache:  zr.cache,
	}, nil
}

func (zr *ZIMReader) readPathPointer(index int) (uint64, error) {
	var buf [8]byte
	if _, err := zr.file.ReadAt(buf[:], int64(zr.header.PathPtrPos)+int64(index)*8); err != nil {
		return 0, fmt.Errorf("failed to read path pointer at %d: %w", index, err)
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}

func (zr *ZIMReader) readClusterPointer(index int) (uint64, error) {
	var buf [8]byte
	if _, err := zr.file.ReadAt(buf[:], int64(zr.header.ClusterPtrPos)+int64(index)*8); err != nil {
		return 0, fmt.Errorf("failed to read cluster pointer at %d: %w", index, err)
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}
