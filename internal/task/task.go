package task

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	TaskStarter interface {
		// Start starts the object that implements TaskStarter,
		// and returns an error if it fails to start.
		//
		// callerSubtask.Finish must be called when start fails or the object is finished.
		Start(parent Parent) E.Error
		Task() *Task
	}
	TaskFinisher interface {
		Finish(reason any)
	}
	// Task controls objects' lifetime.
	//
	// Objects that uses a Task should implement the TaskStarter and the TaskFinisher interface.
	//
	// Use Task.Finish to stop all subtasks of the Task.
	Task struct {
		name string

		children sync.WaitGroup

		onFinished sync.WaitGroup
		finished   chan struct{}

		ctx    context.Context
		cancel context.CancelCauseFunc

		once sync.Once
	}
	Parent interface {
		Context() context.Context
		Subtask(name string, needFinish ...bool) *Task
		Name() string
		Finish(reason any)
		OnCancel(name string, f func())
	}
)

func (t *Task) Context() context.Context {
	return t.ctx
}

func (t *Task) Finished() <-chan struct{} {
	return t.finished
}

// FinishCause returns the reason / error that caused the task to be finished.
func (t *Task) FinishCause() error {
	return context.Cause(t.ctx)
}

// OnFinished calls fn when the task is canceled and all subtasks are finished.
//
// It should not be called after Finish is called.
func (t *Task) OnFinished(about string, fn func()) {
	t.onCancel(about, fn, true)
}

// OnCancel calls fn when the task is canceled.
//
// It should not be called after Finish is called.
func (t *Task) OnCancel(about string, fn func()) {
	t.onCancel(about, fn, false)
}

func (t *Task) onCancel(about string, fn func(), waitSubTasks bool) {
	t.onFinished.Add(1)
	go func() {
		<-t.ctx.Done()
		if waitSubTasks {
			t.children.Wait()
		}
		t.invokeWithRecover(fn, about)
		t.onFinished.Done()
	}()
}

// Finish cancel all subtasks and wait for them to finish,
// then marks the task as finished, with the given reason (if any).
func (t *Task) Finish(reason any) {
	select {
	case <-t.finished:
		return
	default:
		t.once.Do(func() {
			t.finish(reason)
		})
	}
}

func (t *Task) finish(reason any) {
	t.cancel(fmtCause(reason))
	t.children.Wait()
	t.onFinished.Wait()
	if t.finished != nil {
		close(t.finished)
	}
	logger.Trace().Msg("task " + t.name + " finished")
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

// Subtask returns a new subtask with the given name, derived from the parent's context.
//
// This should not be called after Finish is called.
func (t *Task) Subtask(name string, needFinish ...bool) *Task {
	nf := len(needFinish) == 0 || needFinish[0]

	ctx, cancel := context.WithCancelCause(t.ctx)
	child := &Task{
		finished: make(chan struct{}, 1),
		ctx:      ctx,
		cancel:   cancel,
	}
	if t != root {
		child.name = t.name + "." + name
		allTasks.Add(child)
	} else {
		child.name = name
	}

	allTasksWg.Add(1)
	t.children.Add(1)

	if !nf {
		go func() {
			<-child.ctx.Done()
			child.Finish(nil)
		}()
	}

	go func() {
		<-child.finished
		allTasksWg.Done()
		t.children.Done()
		allTasks.Remove(child)
	}()

	logger.Trace().Msg("task " + child.name + " started")
	return child
}

// Name returns the name of the task without parent names.
func (t *Task) Name() string {
	parts := strutils.SplitRune(t.name, '.')
	return parts[len(parts)-1]
}

// String returns the full name of the task.
func (t *Task) String() string {
	return t.name
}

// MarshalText implements encoding.TextMarshaler.
func (t *Task) MarshalText() ([]byte, error) {
	return []byte(t.name), nil
}

func (t *Task) invokeWithRecover(fn func(), caller string) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error().
				Interface("err", err).
				Msg("panic in task " + t.name + "." + caller)
			if common.IsDebug {
				panic(string(debug.Stack()))
			}
		}
	}()
	fn()
}
