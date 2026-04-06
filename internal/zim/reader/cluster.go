package reader

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

func (c *Cluster) ReadBlob(blobIndex uint32) ([]byte, error) {
	data, err := c.readUncompressedData()
	if err != nil {
		return nil, err
	}

	offsetSize := 4
	if c.Extended {
		offsetSize = 8
	}

	if len(data) < 1+offsetSize {
		return nil, fmt.Errorf("cluster data too short to read first offset: len=%d, need=%d", len(data), 1+offsetSize)
	}

	firstOffsetBytes := data[1 : 1+offsetSize]
	var firstOffset uint64
	if c.Extended {
		firstOffset = binary.LittleEndian.Uint64(firstOffsetBytes)
	} else {
		firstOffset = uint64(binary.LittleEndian.Uint32(firstOffsetBytes))
	}

	blobCount := (firstOffset / uint64(offsetSize)) - 1

	if blobIndex >= uint32(blobCount) {
		return nil, fmt.Errorf("blob index %d out of range (max %d)", blobIndex, blobCount)
	}

	offsetPos := 1 + (blobIndex * uint32(offsetSize))
	endPos := offsetPos + uint32(offsetSize)*2
	if int(endPos) > len(data) {
		return nil, fmt.Errorf("cluster data too short to read blob offsets: len=%d, need=%d", len(data), endPos)
	}

	var startOffset, endOffset uint64

	if c.Extended {
		startOffset = binary.LittleEndian.Uint64(data[offsetPos : offsetPos+8])
		endOffset = binary.LittleEndian.Uint64(data[offsetPos+8 : offsetPos+16])
	} else {
		startOffset = uint64(binary.LittleEndian.Uint32(data[offsetPos : offsetPos+4]))
		endOffset = uint64(binary.LittleEndian.Uint32(data[offsetPos+4 : offsetPos+8]))
	}

	startOffset += 1
	endOffset += 1

	if startOffset > endOffset || endOffset > uint64(len(data)) {
		return nil, fmt.Errorf("invalid blob offsets: start=%d, end=%d, dataLen=%d", startOffset, endOffset, len(data))
	}

	return data[startOffset:endOffset], nil
}

func (c *Cluster) BlobReader(blobIndex uint32) (io.ReaderAt, int64, error) {
	clusterInfo := make([]byte, 1)
	if _, err := c.reader.ReadAt(clusterInfo, int64(c.offset)); err != nil {
		return nil, 0, fmt.Errorf("failed to read cluster info: %w", err)
	}

	compression := CompressionType(clusterInfo[0] & 0x0F)
	c.Compression = compression
	c.Extended = (clusterInfo[0] & 0x10) != 0

	if compression != CompressionNone {
		data, err := c.ReadBlob(blobIndex)
		if err != nil {
			return nil, 0, err
		}
		return bytes.NewReader(data), int64(len(data)), nil
	}

	offsetSize := 4
	if c.Extended {
		offsetSize = 8
	}

	firstOffsetBuf := make([]byte, offsetSize)
	if _, err := c.reader.ReadAt(firstOffsetBuf, int64(c.offset+1)); err != nil {
		return nil, 0, fmt.Errorf("failed to read first offset: %w", err)
	}

	var firstOffset uint64
	if c.Extended {
		firstOffset = binary.LittleEndian.Uint64(firstOffsetBuf)
	} else {
		firstOffset = uint64(binary.LittleEndian.Uint32(firstOffsetBuf))
	}

	blobCount := (firstOffset / uint64(offsetSize)) - 1
	if blobIndex >= uint32(blobCount) {
		return nil, 0, fmt.Errorf("blob index %d out of range (max %d)", blobIndex, blobCount)
	}

	offsetsPos := int64(c.offset) + 1 + int64(blobIndex)*int64(offsetSize)
	offsetsBuf := make([]byte, offsetSize*2)
	if _, err := c.reader.ReadAt(offsetsBuf, offsetsPos); err != nil {
		return nil, 0, fmt.Errorf("failed to read blob offsets: %w", err)
	}

	var startOffset, endOffset uint64
	if c.Extended {
		startOffset = binary.LittleEndian.Uint64(offsetsBuf[:8])
		endOffset = binary.LittleEndian.Uint64(offsetsBuf[8:])
	} else {
		startOffset = uint64(binary.LittleEndian.Uint32(offsetsBuf[:4]))
		endOffset = uint64(binary.LittleEndian.Uint32(offsetsBuf[4:]))
	}

	blobStart := int64(c.offset) + 1 + int64(startOffset)
	blobSize := int64(endOffset - startOffset)

	return io.NewSectionReader(c.reader, blobStart, blobSize), blobSize, nil
}

func (c *Cluster) readUncompressedData() ([]byte, error) {
	clusterInfo := make([]byte, 1)
	if _, err := c.reader.ReadAt(clusterInfo, int64(c.offset)); err != nil {
		return nil, fmt.Errorf("failed to read cluster info: %w", err)
	}

	c.Compression = CompressionType(clusterInfo[0] & 0x0F)
	c.Extended = (clusterInfo[0] & 0x10) != 0

	const maxClusterSize = 1 << 30 // 1 GB
	if c.size == 0 || c.size > maxClusterSize {
		return nil, fmt.Errorf("invalid cluster size: %d", c.size)
	}

	compressedData := make([]byte, c.size-1)
	n, err := c.reader.ReadAt(compressedData, int64(c.offset+1))
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read cluster data: %w", err)
	}
	compressedData = compressedData[:n]

	switch c.Compression {
	case CompressionNone, CompressionType(0):
		result := make([]byte, len(compressedData)+1)
		result[0] = clusterInfo[0]
		copy(result[1:], compressedData)
		return result, nil

	case CompressionLZMA2:
		return decompressLZMA2(clusterInfo[0], compressedData)

	case CompressionZstd:
		return decompressZstd(clusterInfo[0], compressedData)

	default:
		return nil, fmt.Errorf("unsupported compression type: %d", c.Compression)
	}
}

func decompressLZMA2(header byte, data []byte) ([]byte, error) {
	reader, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create LZMA2 reader: %w", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, fmt.Errorf("failed to decompress LZMA2: %w", err)
	}

	result := make([]byte, buf.Len()+1)
	result[0] = header
	copy(result[1:], buf.Bytes())
	return result, nil
}

func decompressZstd(header byte, data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create Zstd reader: %w", err)
	}
	defer decoder.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, decoder); err != nil {
		return nil, fmt.Errorf("failed to decompress Zstd: %w", err)
	}

	result := make([]byte, buf.Len()+1)
	result[0] = header
	copy(result[1:], buf.Bytes())
	return result, nil
}
