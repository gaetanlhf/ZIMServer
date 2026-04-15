package reader

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	mimeTypeRedirect       uint16 = 0xffff
	contentEntryFixedSize         = 16
	redirectEntryFixedSize        = 12
	maxStringLen                  = 64 * 1024
)

func readDirectoryEntry(r io.ReaderAt, offset uint64) (DirectoryEntry, error) {
	var header [contentEntryFixedSize]byte
	if _, err := r.ReadAt(header[:], int64(offset)); err != nil {
		return nil, fmt.Errorf("failed to read directory entry: %w", err)
	}

	mimeType := binary.LittleEndian.Uint16(header[0:2])
	extraLen := uint16(header[2])
	namespace := header[3]
	revision := binary.LittleEndian.Uint32(header[4:8])

	if mimeType == mimeTypeRedirect {
		redirectIndex := binary.LittleEndian.Uint32(header[8:12])
		if redirectIndex == 0xffffffff {
			return nil, fmt.Errorf("invalid redirect index 0xffffffff")
		}
		path, title, err := readPathTitle(r, offset+redirectEntryFixedSize, extraLen)
		if err != nil {
			return nil, err
		}
		return &RedirectEntry{
			Namespace:     namespace,
			Revision:      revision,
			RedirectIndex: redirectIndex,
			Path:          path,
			Title:         titleOrPath(title, path),
		}, nil
	}

	if mimeType == 0xfffe || mimeType == 0xfffd {
		return nil, fmt.Errorf("deprecated dirent mime type 0x%x at offset %d", mimeType, offset)
	}

	path, title, err := readPathTitle(r, offset+contentEntryFixedSize, extraLen)
	if err != nil {
		return nil, err
	}
	return &ContentEntry{
		MimeType:      mimeType,
		Namespace:     namespace,
		Revision:      revision,
		ClusterNumber: binary.LittleEndian.Uint32(header[8:12]),
		BlobNumber:    binary.LittleEndian.Uint32(header[12:16]),
		Path:          path,
		Title:         titleOrPath(title, path),
	}, nil
}

func titleOrPath(title, path string) string {
	if title == "" {
		return path
	}
	return title
}

func readPathTitle(r io.ReaderAt, offset uint64, extraLen uint16) (string, string, error) {
	budget := maxStringLen - int(extraLen)
	if budget <= 0 {
		budget = maxStringLen
	}
	path, pathLen, err := readZeroTermString(r, offset, budget)
	if err != nil {
		return "", "", fmt.Errorf("failed to read path: %w", err)
	}
	title, _, err := readZeroTermString(r, offset+uint64(pathLen), budget-pathLen)
	if err != nil {
		return "", "", fmt.Errorf("failed to read title: %w", err)
	}
	return path, title, nil
}

func readZeroTermString(r io.ReaderAt, offset uint64, maxBytes int) (string, int, error) {
	if maxBytes <= 0 {
		maxBytes = maxStringLen
	}
	const chunk = 256
	buf := make([]byte, 0, chunk)
	tmp := make([]byte, chunk)
	pos := offset
	for {
		readSize := chunk
		if remaining := maxBytes - len(buf); remaining < readSize {
			readSize = remaining
		}
		if readSize <= 0 {
			return "", 0, fmt.Errorf("string exceeds %d bytes", maxBytes)
		}
		n, err := r.ReadAt(tmp[:readSize], int64(pos))
		if n == 0 && err != nil {
			return "", 0, err
		}
		for i := 0; i < n; i++ {
			if tmp[i] == 0 {
				buf = append(buf, tmp[:i]...)
				return string(buf), len(buf) + 1, nil
			}
		}
		buf = append(buf, tmp[:n]...)
		if err != nil {
			return "", 0, fmt.Errorf("unterminated string at offset %d", offset)
		}
		pos += uint64(n)
	}
}
