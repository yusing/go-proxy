package task

import (
	"errors"
	"fmt"
	"time"
)

func (t *Task) addCallback(about string, fn func(), waitSubTasks bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.callbacks == nil {
		t.callbacks = make(map[*Callback]struct{})
	}
	if t.callbacksDone == nil {
		t.callbacksDone = make(chan struct{})
	}
	t.callbacks[&Callback{fn, about, waitSubTasks}] = struct{}{}
}

func (t *Task) addChildCount() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.children++
	if t.children == 1 {
		t.childrenDone = make(chan struct{})
	}
}

func (t *Task) subChildCount() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.children--
	switch t.children {
	case 0:
		close(t.childrenDone)
	case ^uint32(0):
		panic("negative child count")
	}
}

func (t *Task) runCallbacks() {
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
