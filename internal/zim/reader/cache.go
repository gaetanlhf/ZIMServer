package reader

import (
	"container/list"
	"sync"
)

const defaultClusterCacheEntries = 64

type clusterCache struct {
	mu       sync.Mutex
	capacity int
	items    map[uint32]*list.Element
	lru      *list.List
}

type clusterCacheNode struct {
	index uint32
	state *clusterState
}

func newClusterCache(capacity int) *clusterCache {
	if capacity <= 0 {
		capacity = defaultClusterCacheEntries
	}
	return &clusterCache{
		capacity: capacity,
		items:    make(map[uint32]*list.Element),
		lru:      list.New(),
	}
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

	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[index]; ok {
		c.lru.MoveToFront(elem)
		return elem.Value.(*clusterCacheNode).state, nil
	}
	elem := c.lru.PushFront(&clusterCacheNode{index: index, state: state})
	c.items[index] = elem
	for c.lru.Len() > c.capacity {
		oldest := c.lru.Back()
		if oldest == nil {
			break
		}
		node := oldest.Value.(*clusterCacheNode)
		node.state.close()
		delete(c.items, node.index)
		c.lru.Remove(oldest)
	}
	return state, nil
}
