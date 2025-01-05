package utils

import (
	"sync/atomic"
)

type RefCount struct {
	_ NoCopy

	refCount uint32
	zeroCh   chan struct{}
}

func NewRefCounter() *RefCount {
	rc := &RefCount{
		refCount: 1,
		zeroCh:   make(chan struct{}),
	}
	return rc
}

func (rc *RefCount) Zero() <-chan struct{} {
	return rc.zeroCh
}

func (rc *RefCount) Add() {
	// We add before checking to ensure proper ordering
	newV := atomic.AddUint32(&rc.refCount, 1)
	if newV == 1 {
		// If it was 0 before we added, that means we're incrementing after a close
		// This is a programming error
		panic("RefCount.Add() called after count reached zero")
	}
}

func (rc *RefCount) Sub() {
	// First read the current value
	for {
		current := atomic.LoadUint32(&rc.refCount)
		if current == 0 {
			// Already at zero, channel should be closed
			return
		}

		// Try to decrement, but only if the value hasn't changed
		if atomic.CompareAndSwapUint32(&rc.refCount, current, current-1) {
			if current == 1 { // Was this the last reference?
				close(rc.zeroCh)
			}
			return
		}
		// If CAS failed, someone else modified the count, try again
	}
}
