package reader

import "sort"

const defaultNarrowDownEntries = 256

type narrowDown struct {
	entries []narrowDownEntry
}

type narrowDownEntry struct {
	pseudoKey string
	lindex    int
}

type narrowDownRange struct {
	begin, end int
}

func (n *narrowDown) add(key string, index int, nextKey string) {
	if len(n.entries) == 0 {
		n.entries = append(n.entries, narrowDownEntry{pseudoKey: key, lindex: index})
		return
	}
	pseudo := shortestStringInBetween(key, nextKey)
	n.entries = append(n.entries, narrowDownEntry{pseudoKey: pseudo, lindex: index})
}

func (n *narrowDown) close(key string, index int) {
	n.entries = append(n.entries, narrowDownEntry{pseudoKey: key, lindex: index})
}

func (n *narrowDown) lookup(key string) narrowDownRange {
	if len(n.entries) == 0 {
		return narrowDownRange{0, 0}
	}
	idx := sort.Search(len(n.entries), func(i int) bool {
		return n.entries[i].pseudoKey > key
	})
	if idx == 0 {
		return narrowDownRange{0, 0}
	}
	prev := n.entries[idx-1].lindex
	if idx == len(n.entries) {
		return narrowDownRange{prev, prev + 1}
	}
	return narrowDownRange{prev, n.entries[idx].lindex + 1}
}

func shortestStringInBetween(a, b string) string {
	min := len(a)
	if len(b) < min {
		min = len(b)
	}
	i := 0
	for i < min && a[i] == b[i] {
		i++
	}
	if i < len(b) {
		return b[:i+1]
	}
	return b
}

func (zr *ZIMReader) ensurePathGrid() {
	zr.pathGridOnce.Do(func() {
		count := int(zr.header.EntryCount)
		if count <= 1 {
			return
		}
		step := count / defaultNarrowDownEntries
		if step < 1 {
			step = 1
		}
		grid := &narrowDown{}
		for i := 0; i < count-1; i += step {
			cur, err := zr.entryAtIndex(i)
			if err != nil {
				return
			}
			next, err := zr.entryAtIndex(i + 1)
			if err != nil {
				return
			}
			grid.add(entryFullPath(cur), i, entryFullPath(next))
		}
		last, err := zr.entryAtIndex(count - 1)
		if err != nil {
			return
		}
		grid.close(entryFullPath(last), count-1)
		zr.pathGrid = grid
	})
}
