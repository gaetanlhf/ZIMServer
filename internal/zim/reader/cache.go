package reader

import (
	"container/list"
	"sync"
	"sync/atomic"
)

const (
	defaultClusterCacheEntries = 16
	defaultClusterCacheBytes   = 128 * 1024 * 1024
)

type clusterCache struct {
	mu       sync.Mutex
	capacity int
	maxBytes int64
	bytes    atomic.Int64
	items    map[uint32]*list.Element
	lru      *list.List
}

type clusterCacheNode struct {
	index uint32
	state *clusterState
}

func newClusterCache(capacity int, maxBytes int64) *clusterCache {
	if capacity <= 0 {
		capacity = defaultClusterCacheEntries
	}
	if maxBytes <= 0 {
		maxBytes = defaultClusterCacheBytes
	}
	c := &clusterCache{
		capacity: capacity,
		maxBytes: maxBytes,
		items:    make(map[uint32]*list.Element),
		lru:      list.New(),
	}
	return c
}

func (c *clusterCache) GetOrBuild(index uint32, factory func() (*clusterState, error)) (*clusterState, error) {
	c.mu.Lock()
	if elem, ok := c.items[index]; ok {
		c.lru.MoveToFront(elem)
		state := elem.Value.(*clusterCacheNode).state
		c.mu.Unlock()
		return state, nil
	}
	c.mu.Unlock()

	state, err := factory()
	if err != nil {
		return nil, err
	}
	state.cache = c

	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[index]; ok {
		c.lru.MoveToFront(elem)
		return elem.Value.(*clusterCacheNode).state, nil
	}
	elem := c.lru.PushFront(&clusterCacheNode{index: index, state: state})
	c.items[index] = elem
	c.evictLocked()
	return state, nil
}

func (c *clusterCache) noteGrowth(delta int64) {
	if delta <= 0 {
		return
	}
	c.bytes.Add(delta)
	if c.bytes.Load() <= c.maxBytes {
		return
	}
	c.mu.Lock()
	c.evictLocked()
	c.mu.Unlock()
}

func (c *clusterCache) evictLocked() {
	for c.lru.Len() > c.capacity || c.bytes.Load() > c.maxBytes {
		oldest := c.lru.Back()
		if oldest == nil {
			return
		}
		node := oldest.Value.(*clusterCacheNode)
		freed := node.state.discard()
		c.bytes.Add(-freed)
		delete(c.items, node.index)
		c.lru.Remove(oldest)
	}
}
