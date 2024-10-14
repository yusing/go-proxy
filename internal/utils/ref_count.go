package utils

type RefCount struct {
	_ NoCopy

	refCh      chan bool
	notifyZero chan struct{}
}

func NewRefCounter() *RefCount {
	rc := &RefCount{
		refCh:      make(chan bool, 1),
		notifyZero: make(chan struct{}),
	}
	go func() {
		refCount := uint32(1)
		for isAdd := range rc.refCh {
			if isAdd {
				refCount++
			} else {
				refCount--
			}
			if refCount <= 0 {
				close(rc.notifyZero)
				return
			}
		}
	}()
	return rc
}

func (rc *RefCount) Zero() <-chan struct{} {
	return rc.notifyZero
}

func (rc *RefCount) Add() {
	rc.refCh <- true
}

func (rc *RefCount) Sub() {
	rc.refCh <- false
}
