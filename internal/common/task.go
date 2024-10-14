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
	globalCtxWg                sync.WaitGroup
	globalCtxTraceMap          = xsync.NewMapOf[*task, struct{}]()
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

func (t *task) Context() context.Context {
	return t.ctx
}

func (t *task) Finished() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.finished {
		return
	}
	t.finished = true
	if _, ok := globalCtxTraceMap.Load(t); ok {
		globalCtxWg.Done()
		globalCtxTraceMap.Delete(t)
	}
}

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

func (t *task) Tree(prefix ...string) string {
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
		sb.WriteString(sub.Tree(pre + "  "))
	}
	return sb.String()
}

func newSubTask(ctx context.Context, name string) *task {
	t := &task{
		ctx:  ctx,
		name: name,
	}
	globalCtxTraceMap.Store(t, struct{}{})
	globalCtxWg.Add(1)
	return t
}

func NewTask(format string, args ...interface{}) Task {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	return newSubTask(globalCtx, format)
}

func NewTaskWithCancel(format string, args ...interface{}) (Task, context.CancelFunc) {
	subCtx, cancel := context.WithCancel(globalCtx)
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	return newSubTask(subCtx, format), cancel
}

func GlobalTask(format string, args ...interface{}) Task {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	return &task{
		ctx:  globalCtx,
		name: format,
	}
}

func CancelGlobalContext() {
	globalCtxCancel()
}

func GlobalContextWait(timeout time.Duration) {
	done := make(chan struct{})
	after := time.After(timeout)
	go func() {
		globalCtxWg.Wait()
		close(done)
	}()
	for {
		select {
		case <-done:
			return
		case <-after:
			logrus.Println("Timeout waiting for these tasks to finish:")
			globalCtxTraceMap.Range(func(t *task, _ struct{}) bool {
				logrus.Println(t.Tree())
				return true
			})
			return
		}
	}
}
