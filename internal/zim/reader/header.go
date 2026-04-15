package reader

import (
	"encoding/binary"
	"fmt"
	"io"
)

const headerSize = 80

func readHeader(r io.ReaderAt) (*Header, error) {
	var buf [headerSize]byte
	if _, err := r.ReadAt(buf[:], 0); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	h := &Header{
		MagicNumber:   binary.LittleEndian.Uint32(buf[0:4]),
		MajorVersion:  binary.LittleEndian.Uint16(buf[4:6]),
		MinorVersion:  binary.LittleEndian.Uint16(buf[6:8]),
		EntryCount:    binary.LittleEndian.Uint32(buf[24:28]),
		ClusterCount:  binary.LittleEndian.Uint32(buf[28:32]),
		PathPtrPos:    binary.LittleEndian.Uint64(buf[32:40]),
		TitlePtrPos:   binary.LittleEndian.Uint64(buf[40:48]),
		ClusterPtrPos: binary.LittleEndian.Uint64(buf[48:56]),
		MimeListPos:   binary.LittleEndian.Uint64(buf[56:64]),
		MainPage:      binary.LittleEndian.Uint32(buf[64:68]),
		LayoutPage:    binary.LittleEndian.Uint32(buf[68:72]),
		ChecksumPos:   binary.LittleEndian.Uint64(buf[72:80]),
	}
	copy(h.UUID[:], buf[8:24])

	if h.MagicNumber != MagicNumber {
		return nil, fmt.Errorf("invalid magic number: got 0x%x, expected 0x%x", h.MagicNumber, MagicNumber)
	}
	if h.MajorVersion != 6 || h.MinorVersion < 1 {
		return nil, fmt.Errorf("unsupported ZIM version %d.%d (only v6.1+ is supported)", h.MajorVersion, h.MinorVersion)
	}
	if h.MimeListPos != headerSize {
		return nil, fmt.Errorf("invalid header: MimeListPos %d (expected %d)", h.MimeListPos, headerSize)
	}
	if h.PathPtrPos < h.MimeListPos {
		return nil, fmt.Errorf("invalid header: PathPtrPos %d precedes MimeListPos", h.PathPtrPos)
	}
	if h.ClusterPtrPos < h.MimeListPos {
		return nil, fmt.Errorf("invalid header: ClusterPtrPos %d precedes MimeListPos", h.ClusterPtrPos)
	}
	if h.ChecksumPos != 0 && h.ChecksumPos < h.MimeListPos {
		return nil, fmt.Errorf("invalid header: ChecksumPos %d precedes MimeListPos", h.ChecksumPos)
	}
	if h.ClusterCount > h.EntryCount {
		return nil, fmt.Errorf("invalid header: ClusterCount %d exceeds EntryCount %d", h.ClusterCount, h.EntryCount)
	}
	if (h.EntryCount == 0) != (h.ClusterCount == 0) {
		return nil, fmt.Errorf("invalid header: EntryCount and ClusterCount must both be zero or non-zero")
	}

	return h, nil
}

func readMimeTypes(r io.ReaderAt, pos, limit uint64) ([]string, error) {
	const maxEntries = 1 << 16
	var mimeTypes []string
	offset := pos
	buf := make([]byte, 1024)

	for {
		if offset >= limit {
			return nil, fmt.Errorf("mime list overran section limit")
		}
		n, err := r.ReadAt(buf, int64(offset))
		if err != nil && n == 0 {
			return nil, fmt.Errorf("failed to read mime list: %w", err)
		}
		end := 0
		for end < n && buf[end] != 0 {
			end++
		}
		if end == 0 {
			return mimeTypes, nil
		}
		if end == n {
			return nil, fmt.Errorf("mime entry not terminated at offset %d", offset)
		}
		if len(mimeTypes) >= maxEntries {
			return nil, fmt.Errorf("mime list exceeds %d entries", maxEntries)
		}
		mimeTypes = append(mimeTypes, string(buf[:end]))
		offset += uint64(end + 1)
	}
}
