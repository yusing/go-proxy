package utils

import (
	"sync"
	"sync/atomic"
)

type RefCount struct {
	_ NoCopy

	mu       sync.Mutex
	cond     *sync.Cond
	refCount uint32
	zeroCh   chan struct{}
}

func NewRefCounter() *RefCount {
	rc := &RefCount{
		refCount: 1,
		zeroCh:   make(chan struct{}),
	}
	rc.cond = sync.NewCond(&rc.mu)
	return rc
}

func (rc *RefCount) Zero() <-chan struct{} {
	return rc.zeroCh
}

func (rc *RefCount) Add() {
	atomic.AddUint32(&rc.refCount, 1)
}

func (rc *RefCount) Sub() {
	if atomic.AddUint32(&rc.refCount, ^uint32(0)) == 0 {
		rc.mu.Lock()
		close(rc.zeroCh)
		rc.cond.Broadcast()
		rc.mu.Unlock()
	}
}
