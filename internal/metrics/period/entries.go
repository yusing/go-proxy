package period

import (
	"time"
)

type Entries[T any] struct {
	entries  [maxEntries]*T
	index    int
	count    int
	interval time.Duration
	lastAdd  time.Time
}

const maxEntries = 200

func newEntries[T any](duration time.Duration) *Entries[T] {
	return &Entries[T]{
		interval: duration / maxEntries,
		lastAdd:  time.Now(),
	}
}

func (e *Entries[T]) Add(now time.Time, info *T) {
	if now.Sub(e.lastAdd) < e.interval {
		return
	}
	e.entries[e.index] = info
	e.index++
	if e.index >= maxEntries {
		e.index = 0
	}
	if e.count < maxEntries {
		e.count++
	}
	e.lastAdd = now
}

func (e *Entries[T]) Get() []*T {
	if e.count < maxEntries {
		return e.entries[:e.count]
	}
	res := make([]*T, maxEntries)
	copy(res, e.entries[e.index:])
	copy(res[maxEntries-e.index:], e.entries[:e.index])
	return res
}
