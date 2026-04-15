package reader

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

type clusterState struct {
	mu          sync.Mutex
	cluster     *Cluster
	cache       *clusterCache
	offsets     []uint64
	decoder     io.ReadCloser
	decoded     []byte
	decodedUpto uint64
	offsetsRead bool
}

func (s *clusterState) discard() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	freed := int64(len(s.decoded))
	if s.decoder != nil {
		s.decoder.Close()
		s.decoder = nil
	}
	s.decoded = nil
	s.decodedUpto = 0
	s.offsets = nil
	s.offsetsRead = false
	s.cache = nil
	return freed
}

func (s *clusterState) noteGrowth(delta int64) {
	if delta > 0 && s.cache != nil {
		s.cache.noteGrowth(delta)
	}
}

func newClusterState(c *Cluster) *clusterState {
	return &clusterState{cluster: c}
}

func (s *clusterState) close() {
	if s.decoder != nil {
		s.decoder.Close()
		s.decoder = nil
	}
}

func (s *clusterState) openDecoder() (io.ReadCloser, error) {
	src := io.NewSectionReader(s.cluster.reader, int64(s.cluster.offset)+1, int64(s.cluster.size-1))
	switch s.cluster.Compression {
	case CompressionNone:
		return io.NopCloser(src), nil
	case CompressionLZMA2:
		dec, err := xz.NewReader(src)
		if err != nil {
			return nil, fmt.Errorf("failed to create LZMA2 reader: %w", err)
		}
		return io.NopCloser(dec), nil
	case CompressionZstd:
		dec, err := zstd.NewReader(src)
		if err != nil {
			return nil, fmt.Errorf("failed to create Zstd reader: %w", err)
		}
		return zstdCloser{dec}, nil
	default:
		return nil, fmt.Errorf("unsupported compression type %d", s.cluster.Compression)
	}
}

type zstdCloser struct{ *zstd.Decoder }

func (z zstdCloser) Close() error { z.Decoder.Close(); return nil }

func (s *clusterState) ensureOffsets() error {
	if s.offsetsRead {
		return nil
	}

	offsetSize := s.cluster.offsetSize()

	if s.cluster.Compression == CompressionNone {
		head := make([]byte, offsetSize)
		if _, err := s.cluster.reader.ReadAt(head, int64(s.cluster.offset)+1); err != nil {
			return fmt.Errorf("failed to read first offset: %w", err)
		}
		first := decodeOffset(head, s.cluster.Extended)
		if err := validateFirstOffset(first, offsetSize); err != nil {
			return err
		}
		count := int(first/uint64(offsetSize)) - 1
		table := make([]byte, (count+1)*offsetSize)
		if _, err := s.cluster.reader.ReadAt(table, int64(s.cluster.offset)+1); err != nil {
			return fmt.Errorf("failed to read offset table: %w", err)
		}
		offsets, err := parseOffsetTable(table, count+1, s.cluster.Extended)
		if err != nil {
			return err
		}
		s.offsets = offsets
		s.offsetsRead = true
		return nil
	}

	if s.decoder == nil {
		dec, err := s.openDecoder()
		if err != nil {
			return err
		}
		s.decoder = dec
	}

	head := make([]byte, offsetSize)
	if _, err := io.ReadFull(s.decoder, head); err != nil {
		s.resetDecoder()
		return fmt.Errorf("failed to read first offset: %w", err)
	}
	s.decoded = append(s.decoded, head...)
	s.decodedUpto = uint64(offsetSize)
	s.noteGrowth(int64(offsetSize))

	first := decodeOffset(head, s.cluster.Extended)
	if err := validateFirstOffset(first, offsetSize); err != nil {
		s.resetDecoder()
		return err
	}
	count := int(first/uint64(offsetSize)) - 1
	tail := make([]byte, count*offsetSize)
	if _, err := io.ReadFull(s.decoder, tail); err != nil {
		s.resetDecoder()
		return fmt.Errorf("failed to read offset table tail: %w", err)
	}
	s.decoded = append(s.decoded, tail...)
	s.decodedUpto += uint64(len(tail))
	s.noteGrowth(int64(len(tail)))

	full := append(append([]byte{}, head...), tail...)
	offsets, err := parseOffsetTable(full, count+1, s.cluster.Extended)
	if err != nil {
		s.resetDecoder()
		return err
	}
	s.offsets = offsets
	s.offsetsRead = true
	return nil
}

func (s *clusterState) blobBounds(idx uint32) (uint64, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOffsets(); err != nil {
		return 0, 0, err
	}
	if int(idx)+1 >= len(s.offsets) {
		return 0, 0, fmt.Errorf("blob %d out of range (max %d)", idx, len(s.offsets)-1)
	}
	start, end := s.offsets[idx], s.offsets[idx+1]
	if start > end {
		return 0, 0, fmt.Errorf("invalid blob offsets: start=%d end=%d", start, end)
	}
	return start, end, nil
}

func (s *clusterState) readBlob(idx uint32) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureOffsets(); err != nil {
		return nil, err
	}
	if int(idx)+1 >= len(s.offsets) {
		return nil, fmt.Errorf("blob %d out of range (max %d)", idx, len(s.offsets)-1)
	}
	start, end := s.offsets[idx], s.offsets[idx+1]
	if start > end {
		return nil, fmt.Errorf("invalid blob offsets: start=%d end=%d", start, end)
	}
	size := end - start

	if s.cluster.Compression == CompressionNone {
		buf := make([]byte, size)
		if size > 0 {
			if _, err := s.cluster.reader.ReadAt(buf, int64(s.cluster.offset)+1+int64(start)); err != nil {
				return nil, fmt.Errorf("failed to read blob %d: %w", idx, err)
			}
		}
		return buf, nil
	}

	if err := s.advanceTo(end); err != nil {
		return nil, err
	}
	out := make([]byte, size)
	copy(out, s.decoded[start:end])
	return out, nil
}

func (s *clusterState) advanceTo(target uint64) error {
	if target <= s.decodedUpto {
		return nil
	}
	need := target - s.decodedUpto
	buf := make([]byte, need)
	if _, err := io.ReadFull(s.decoder, buf); err != nil {
		s.resetDecoder()
		return fmt.Errorf("failed to advance decoder: %w", err)
	}
	s.decoded = append(s.decoded, buf...)
	s.decodedUpto = target
	s.noteGrowth(int64(need))
	return nil
}

func (s *clusterState) resetDecoder() {
	freed := int64(len(s.decoded))
	if s.decoder != nil {
		s.decoder.Close()
		s.decoder = nil
	}
	s.decoded = nil
	s.decodedUpto = 0
	s.offsets = nil
	s.offsetsRead = false
	if freed > 0 && s.cache != nil {
		s.cache.bytes.Add(-freed)
	}
}

func decodeOffset(buf []byte, extended bool) uint64 {
	if extended {
		return binary.LittleEndian.Uint64(buf[:8])
	}
	return uint64(binary.LittleEndian.Uint32(buf[:4]))
}

func validateFirstOffset(first uint64, offsetSize int) error {
	if first != uint64(offsetSize) && first < uint64(2*offsetSize) {
		return fmt.Errorf("first offset too small: %d", first)
	}
	if first%uint64(offsetSize) != 0 {
		return fmt.Errorf("first offset not aligned: %d", first)
	}
	return nil
}

func parseOffsetTable(buf []byte, count int, extended bool) ([]uint64, error) {
	offsets := make([]uint64, count)
	step := 4
	if extended {
		step = 8
	}
	for i := 0; i < count; i++ {
		if extended {
			offsets[i] = binary.LittleEndian.Uint64(buf[i*step : i*step+8])
		} else {
			offsets[i] = uint64(binary.LittleEndian.Uint32(buf[i*step : i*step+4]))
		}
		if i > 0 && offsets[i] < offsets[i-1] {
			return nil, fmt.Errorf("blob offsets not monotonic at %d: %d < %d", i, offsets[i], offsets[i-1])
		}
	}
	return offsets, nil
}
