package reader

import (
	"container/list"
	"sync"
	"sync/atomic"
)

const (
	defaultClusterCacheEntries = 16
	defaultClusterCacheBytes   = 128 * 1024 * 1024
	clusterCacheShards         = 16
)

type clusterCacheShard struct {
	mu    sync.Mutex
	items map[uint32]*list.Element
	lru   *list.List
}

type clusterCache struct {
	shards         [clusterCacheShards]clusterCacheShard
	capacityShard  int
	maxBytes       int64
	bytes          atomic.Int64
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
		capacityShard: (capacity + clusterCacheShards - 1) / clusterCacheShards,
		maxBytes:      maxBytes,
	}
	for i := range c.shards {
		c.shards[i].items = make(map[uint32]*list.Element)
		c.shards[i].lru = list.New()
	}
	return c
}

func (c *clusterCache) shardFor(index uint32) *clusterCacheShard {
	return &c.shards[index%clusterCacheShards]
}

func (c *clusterCache) GetOrBuild(index uint32, factory func() (*clusterState, error)) (*clusterState, error) {
	sh := c.shardFor(index)
	sh.mu.Lock()
	if elem, ok := sh.items[index]; ok {
		sh.lru.MoveToFront(elem)
		state := elem.Value.(*clusterCacheNode).state
		sh.mu.Unlock()
		return state, nil
	}
	sh.mu.Unlock()

	state, err := factory()
	if err != nil {
		return nil, err
	}
	state.cache = c

	sh.mu.Lock()
	defer sh.mu.Unlock()
	if elem, ok := sh.items[index]; ok {
		sh.lru.MoveToFront(elem)
		return elem.Value.(*clusterCacheNode).state, nil
	}
	elem := sh.lru.PushFront(&clusterCacheNode{index: index, state: state})
	sh.items[index] = elem
	c.evictShardLocked(sh)
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
	c.evictGlobal()
}

func (c *clusterCache) evictGlobal() {
	for i := range c.shards {
		if c.bytes.Load() <= c.maxBytes {
			return
		}
		sh := &c.shards[i]
		sh.mu.Lock()
		c.evictShardLocked(sh)
		sh.mu.Unlock()
	}
}

func (c *clusterCache) evictShardLocked(sh *clusterCacheShard) {
	for sh.lru.Len() > c.capacityShard || c.bytes.Load() > c.maxBytes {
		oldest := sh.lru.Back()
		if oldest == nil {
			return
		}
		node := oldest.Value.(*clusterCacheNode)
		freed := node.state.discard()
		c.bytes.Add(-freed)
		delete(sh.items, node.index)
		sh.lru.Remove(oldest)
	}
}
