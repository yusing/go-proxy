package task

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

func (t *Task) addCallback(about string, fn func(), waitSubTasks bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.callbacks == nil {
		t.callbacks = make(map[*Callback]struct{})
		t.callbacksDone = make(chan struct{})
	}
	t.callbacks[&Callback{fn, about, waitSubTasks}] = struct{}{}
}

func (t *Task) addChildCount() {
	if atomic.AddUint32(&t.children, 1) == 1 {
		t.mu.Lock()
		if t.childrenDone == nil {
			t.childrenDone = make(chan struct{})
		}
		t.mu.Unlock()
	}
}

func (t *Task) subChildCount() {
	if atomic.AddUint32(&t.children, ^uint32(0)) == 0 {
		close(t.childrenDone)
	}
}

func (t *Task) runCallbacks() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.callbacks) == 0 {
		return
	}
	for c := range t.callbacks {
		if c.waitChildren {
			waitWithTimeout(t.childrenDone)
		}
		t.invokeWithRecover(c.fn, c.about)
		delete(t.callbacks, c)
	}
	close(t.callbacksDone)
}

func waitWithTimeout(ch <-chan struct{}) bool {
	if ch == nil {
		return true
	}
	select {
	case <-ch:
		return true
	case <-time.After(taskTimeout):
		return false
	}
}

func fmtCause(cause any) error {
	switch cause := cause.(type) {
	case nil:
		return nil
	case error:
		return cause
	case string:
		return errors.New(cause)
	default:
		return fmt.Errorf("%v", cause)
	}
}
