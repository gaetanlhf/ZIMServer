package reader

import (
	"container/list"
	"sync"
)

const defaultDirentCacheEntries = 2048

type direntCache struct {
	mu       sync.Mutex
	capacity int
	items    map[int]*list.Element
	lru      *list.List
}

type direntCacheNode struct {
	index int
	entry DirectoryEntry
}

func newDirentCache(capacity int) *direntCache {
	if capacity <= 0 {
		capacity = defaultDirentCacheEntries
	}
	return &direntCache{
		capacity: capacity,
		items:    make(map[int]*list.Element),
		lru:      list.New(),
	}
}

func (c *direntCache) Get(index int) (DirectoryEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[index]; ok {
		c.lru.MoveToFront(elem)
		return elem.Value.(*direntCacheNode).entry, true
	}
	return nil, false
}

func (c *direntCache) Put(index int, entry DirectoryEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[index]; ok {
		c.lru.MoveToFront(elem)
		elem.Value.(*direntCacheNode).entry = entry
		return
	}
	elem := c.lru.PushFront(&direntCacheNode{index: index, entry: entry})
	c.items[index] = elem
	for c.lru.Len() > c.capacity {
		oldest := c.lru.Back()
		if oldest == nil {
			return
		}
		delete(c.items, oldest.Value.(*direntCacheNode).index)
		c.lru.Remove(oldest)
	}
}
