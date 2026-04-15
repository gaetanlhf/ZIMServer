package reader

import (
	"bytes"
	"fmt"
	"io"
)

func (c *Cluster) offsetSize() int {
	if c.Extended {
		return 8
	}
	return 4
}

func (c *Cluster) state() (*clusterState, error) {
	if c.cache == nil {
		if err := c.loadHeader(); err != nil {
			return nil, err
		}
		return newClusterState(c), nil
	}
	return c.cache.GetOrBuild(c.index, func() (*clusterState, error) {
		if err := c.loadHeader(); err != nil {
			return nil, err
		}
		return newClusterState(c), nil
	})
}

func (c *Cluster) loadHeader() error {
	var buf [1]byte
	if _, err := c.reader.ReadAt(buf[:], int64(c.offset)); err != nil {
		return fmt.Errorf("failed to read cluster info: %w", err)
	}
	c.Compression = CompressionType(buf[0] & 0x0F)
	c.Extended = (buf[0] & 0x10) != 0
	return nil
}

func (c *Cluster) ReadBlob(blobIndex uint32) ([]byte, error) {
	state, err := c.state()
	if err != nil {
		return nil, err
	}
	return state.readBlob(blobIndex)
}

func (c *Cluster) BlobReader(blobIndex uint32) (io.ReaderAt, int64, error) {
	state, err := c.state()
	if err != nil {
		return nil, 0, err
	}
	if state.cluster.Compression == CompressionNone {
		start, end, err := state.blobBounds(blobIndex)
		if err != nil {
			return nil, 0, err
		}
		size := int64(end - start)
		offset := int64(state.cluster.offset) + 1 + int64(start)
		return io.NewSectionReader(state.cluster.reader, offset, size), size, nil
	}
	data, err := state.readBlob(blobIndex)
	if err != nil {
		return nil, 0, err
	}
	return bytes.NewReader(data), int64(len(data)), nil
}
