package reader

import (
	"encoding/binary"
	"fmt"
	"io"
)

func readHeader(r io.ReaderAt) (*Header, error) {
	buf := make([]byte, 80)
	if _, err := r.ReadAt(buf, 0); err != nil {
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

	if h.MajorVersion < 6 {
		return nil, fmt.Errorf("unsupported ZIM version: %d.%d (minimum required version is 6.0)", h.MajorVersion, h.MinorVersion)
	}

	return h, nil
}

func readMimeTypes(r io.ReaderAt, pos uint64) ([]string, error) {
	var mimeTypes []string
	offset := pos
	buf := make([]byte, 256)

	for {
		if _, err := r.ReadAt(buf, int64(offset)); err != nil {
			return nil, fmt.Errorf("failed to read mime type: %w", err)
		}

		end := 0
		for end < len(buf) && buf[end] != 0 {
			end++
		}

		if end == 0 {
			break
		}

		mimeTypes = append(mimeTypes, string(buf[:end]))
		offset += uint64(end + 1)
	}

	return mimeTypes, nil
}

func readPathPointers(r io.ReaderAt, pos uint64, count uint32) ([]uint64, error) {
	buf := make([]byte, count*8)
	if _, err := r.ReadAt(buf, int64(pos)); err != nil {
		return nil, fmt.Errorf("failed to read path pointers: %w", err)
	}

	pointers := make([]uint64, count)
	for i := uint32(0); i < count; i++ {
		pointers[i] = binary.LittleEndian.Uint64(buf[i*8 : (i+1)*8])
	}

	return pointers, nil
}

func readTitlePointers(r io.ReaderAt, pos uint64, count uint32) ([]uint32, error) {
	buf := make([]byte, count*4)
	if _, err := r.ReadAt(buf, int64(pos)); err != nil {
		return nil, fmt.Errorf("failed to read title pointers: %w", err)
	}

	pointers := make([]uint32, count)
	for i := uint32(0); i < count; i++ {
		pointers[i] = binary.LittleEndian.Uint32(buf[i*4 : (i+1)*4])
	}

	return pointers, nil
}

func readClusterPointers(r io.ReaderAt, pos uint64, count uint32) ([]uint64, error) {
	buf := make([]byte, count*8)
	if _, err := r.ReadAt(buf, int64(pos)); err != nil {
		return nil, fmt.Errorf("failed to read cluster pointers: %w", err)
	}

	pointers := make([]uint64, count)
	for i := uint32(0); i < count; i++ {
		pointers[i] = binary.LittleEndian.Uint64(buf[i*8 : (i+1)*8])
	}

	return pointers, nil
}
