package common

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/sirupsen/logrus"
)

var (
	globalCtx, globalCtxCancel = context.WithCancel(context.Background())
	taskWg                     sync.WaitGroup
	tasksMap                   = xsync.NewMapOf[*task, struct{}]()
)

type (
	Task interface {
		Name() string
		Context() context.Context
		Subtask(usageFmt string, args ...interface{}) Task
		SubtaskWithCancel(usageFmt string, args ...interface{}) (Task, context.CancelFunc)
		Finished()
	}
	task struct {
		ctx      context.Context
		subtasks []*task
		name     string
		finished bool
		mu       sync.Mutex
	}
)

func (t *task) Name() string {
	return t.name
}

// Context returns the context associated with the task. This context is
// canceled when the task is finished.
func (t *task) Context() context.Context {
	return t.ctx
}

// Finished marks the task as finished and notifies the global wait group.
// Finished is thread-safe and idempotent.
func (t *task) Finished() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.finished {
		return
	}
	t.finished = true
	if _, ok := tasksMap.Load(t); ok {
		taskWg.Done()
		tasksMap.Delete(t)
	}
	logrus.Debugf("task %q finished", t.Name())
}

// Subtask returns a new subtask with the given name, derived from the receiver's context.
//
// The returned subtask is associated with the receiver's context and will be
// automatically registered and deregistered from the global task wait group.
//
// If the receiver's context is already canceled, the returned subtask will be
// canceled immediately.
//
// The returned subtask is safe for concurrent use.
func (t *task) Subtask(format string, args ...interface{}) Task {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	sub := newSubTask(t.ctx, format)
	t.subtasks = append(t.subtasks, sub)
	return sub
}

// SubtaskWithCancel returns a new subtask with the given name, derived from the receiver's context,
// and a cancel function. The returned subtask is associated with the receiver's context and will be
// automatically registered and deregistered from the global task wait group.
//
// If the receiver's context is already canceled, the returned subtask will be canceled immediately.
//
// The returned cancel function is safe for concurrent use, and can be used to cancel the returned
// subtask at any time.
func (t *task) SubtaskWithCancel(format string, args ...interface{}) (Task, context.CancelFunc) {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	ctx, cancel := context.WithCancel(t.ctx)
	sub := newSubTask(ctx, format)
	t.subtasks = append(t.subtasks, sub)
	return sub, cancel
}

func (t *task) tree(prefix ...string) string {
	var sb strings.Builder
	var pre string
	if len(prefix) > 0 {
		pre = prefix[0]
	}
	sb.WriteString(pre)
	sb.WriteString(t.Name() + "\n")
	for _, sub := range t.subtasks {
		if sub.finished {
			continue
		}
		sb.WriteString(sub.tree(pre + "  "))
	}
	return sb.String()
}

func newSubTask(ctx context.Context, name string) *task {
	t := &task{
		ctx:  ctx,
		name: name,
	}
	tasksMap.Store(t, struct{}{})
	taskWg.Add(1)
	return t
}

// NewTask returns a new Task with the given name, derived from the global
// context.
//
// The returned Task is associated with the global context and will be
// automatically registered and deregistered from the global context's wait
// group.
//
// If the global context is already canceled, the returned Task will be
// canceled immediately.
//
// The returned Task is not safe for concurrent use.
func NewTask(format string, args ...interface{}) Task {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	return newSubTask(globalCtx, format)
}

// NewTaskWithCancel returns a new Task with the given name, derived from the
// global context, and a cancel function. The returned Task is associated with
// the global context and will be automatically registered and deregistered
// from the global task wait group.
//
// If the global context is already canceled, the returned Task will be
// canceled immediately.
//
// The returned Task is safe for concurrent use.
//
// The returned cancel function is safe for concurrent use, and can be used
// to cancel the returned Task at any time.
func NewTaskWithCancel(format string, args ...interface{}) (Task, context.CancelFunc) {
	subCtx, cancel := context.WithCancel(globalCtx)
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	return newSubTask(subCtx, format), cancel
}

// GlobalTask returns a new Task with the given name, associated with the
// global context.
//
// Unlike NewTask, GlobalTask does not automatically register or deregister
// the Task with the global task wait group. The returned Task is not
// started, but the name is formatted immediately.
//
// This is best used for main task that do not need to wait and
// will create a bunch of subtasks.
//
// The returned Task is safe for concurrent use.
func GlobalTask(format string, args ...interface{}) Task {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	return &task{
		ctx:  globalCtx,
		name: format,
	}
}

// CancelGlobalContext cancels the global context, which will cause all tasks
// created by GlobalTask or NewTask to be canceled. This should be called
// before exiting the program to ensure that all tasks are properly cleaned
// up.
func CancelGlobalContext() {
	globalCtxCancel()
}

// GlobalContextWait waits for all tasks to finish, up to the given timeout.
//
// If the timeout is exceeded, it prints a list of all tasks that were
// still running when the timeout was reached, and their current tree
// of subtasks.
func GlobalContextWait(timeout time.Duration) {
	done := make(chan struct{})
	after := time.After(timeout)
	go func() {
		taskWg.Wait()
		close(done)
	}()
	for {
		select {
		case <-done:
			return
		case <-after:
			logrus.Println("Timeout waiting for these tasks to finish:")
			tasksMap.Range(func(t *task, _ struct{}) bool {
				logrus.Println(t.tree())
				return true
			})
			return
		}
	}
}
