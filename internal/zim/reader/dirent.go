package reader

import (
	"encoding/binary"
	"fmt"
	"io"
)

func readDirectoryEntry(r io.ReaderAt, offset uint64) (DirectoryEntry, error) {
	buf := make([]byte, 256)
	if _, err := r.ReadAt(buf, int64(offset)); err != nil {
		return nil, fmt.Errorf("failed to read directory entry: %w", err)
	}

	mimeType := binary.LittleEndian.Uint16(buf[0:2])
	namespace := buf[3]

	if mimeType == 0xffff {
		return readRedirectEntry(buf, namespace)
	}

	if mimeType == 0xfffe || mimeType == 0xfffd {
		return nil, fmt.Errorf("deprecated entry type: 0x%x", mimeType)
	}

	return readContentEntry(r, buf, offset, mimeType, namespace)
}

func readContentEntry(r io.ReaderAt, buf []byte, offset uint64, mimeType uint16, namespace byte) (*ContentEntry, error) {
	entry := &ContentEntry{
		MimeType:      mimeType,
		Namespace:     namespace,
		Revision:      binary.LittleEndian.Uint32(buf[4:8]),
		ClusterNumber: binary.LittleEndian.Uint32(buf[8:12]),
		BlobNumber:    binary.LittleEndian.Uint32(buf[12:16]),
	}

	path, pathLen, err := readZeroTermString(r, offset+16)
	if err != nil {
		return nil, fmt.Errorf("failed to read path: %w", err)
	}
	entry.Path = path

	title, _, err := readZeroTermString(r, offset+16+uint64(pathLen))
	if err != nil {
		return nil, fmt.Errorf("failed to read title: %w", err)
	}

	if title == "" || title == "null" {
		entry.Title = path
	} else {
		entry.Title = title
	}

	return entry, nil
}

func readRedirectEntry(buf []byte, namespace byte) (*RedirectEntry, error) {
	entry := &RedirectEntry{
		Namespace:     namespace,
		Revision:      binary.LittleEndian.Uint32(buf[4:8]),
		RedirectIndex: binary.LittleEndian.Uint32(buf[8:12]),
	}

	if entry.RedirectIndex == 0xffffffff {
		return nil, fmt.Errorf("invalid redirect index: 0xffffffff")
	}

	pathStart := 12
	pathEnd := pathStart
	for pathEnd < len(buf) && buf[pathEnd] != 0 {
		pathEnd++
	}
	entry.Path = string(buf[pathStart:pathEnd])

	titleStart := pathEnd + 1
	titleEnd := titleStart
	for titleEnd < len(buf) && buf[titleEnd] != 0 {
		titleEnd++
	}
	title := string(buf[titleStart:titleEnd])

	if title == "" || title == "null" {
		entry.Title = entry.Path
	} else {
		entry.Title = title
	}

	return entry, nil
}

func readZeroTermString(r io.ReaderAt, offset uint64) (string, int, error) {
	buf := make([]byte, 512)
	if _, err := r.ReadAt(buf, int64(offset)); err != nil {
		return "", 0, err
	}

	end := 0
	for end < len(buf) && buf[end] != 0 {
		end++
	}

	if end == len(buf) {
		return "", 0, fmt.Errorf("string too long or not null-terminated")
	}

	return string(buf[:end]), end + 1, nil
}
