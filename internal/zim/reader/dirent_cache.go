package reader

import (
	"container/list"
	"sync"
)

const (
	defaultDirentCacheEntries = 2048
	direntCacheShards         = 16
)

type direntCacheShard struct {
	mu    sync.Mutex
	items map[int]*list.Element
	lru   *list.List
}

type direntCache struct {
	shards        [direntCacheShards]direntCacheShard
	capacityShard int
}

type direntCacheNode struct {
	index int
	entry DirectoryEntry
}

func newDirentCache(capacity int) *direntCache {
	if capacity <= 0 {
		capacity = defaultDirentCacheEntries
	}
	c := &direntCache{
		capacityShard: (capacity + direntCacheShards - 1) / direntCacheShards,
	}
	for i := range c.shards {
		c.shards[i].items = make(map[int]*list.Element)
		c.shards[i].lru = list.New()
	}
	return c
}

func (c *direntCache) shardFor(index int) *direntCacheShard {
	if index < 0 {
		index = -index
	}
	return &c.shards[index%direntCacheShards]
}

func (c *direntCache) Get(index int) (DirectoryEntry, bool) {
	sh := c.shardFor(index)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if elem, ok := sh.items[index]; ok {
		sh.lru.MoveToFront(elem)
		return elem.Value.(*direntCacheNode).entry, true
	}
	return nil, false
}

func (c *direntCache) Put(index int, entry DirectoryEntry) {
	sh := c.shardFor(index)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if elem, ok := sh.items[index]; ok {
		sh.lru.MoveToFront(elem)
		elem.Value.(*direntCacheNode).entry = entry
		return
	}
	elem := sh.lru.PushFront(&direntCacheNode{index: index, entry: entry})
	sh.items[index] = elem
	for sh.lru.Len() > c.capacityShard {
		oldest := sh.lru.Back()
		if oldest == nil {
			return
		}
		delete(sh.items, oldest.Value.(*direntCacheNode).index)
		sh.lru.Remove(oldest)
	}
}
