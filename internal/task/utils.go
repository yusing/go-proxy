package task

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/yusing/go-proxy/internal/logging"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

var ErrProgramExiting = errors.New("program exiting")

var logger = logging.With().Str("module", "task").Logger()

var root = newRoot()
var allTasks = F.NewSet[*Task]()
var allTasksWg sync.WaitGroup

func testCleanup() {
	root = newRoot()
	allTasks.Clear()
	allTasksWg = sync.WaitGroup{}
}

// RootTask returns a new Task with the given name, derived from the root context.
func RootTask(name string, needFinish bool) *Task {
	return root.Subtask(name, needFinish)
}

func newRoot() *Task {
	t := &Task{name: "root"}
	t.ctx, t.cancel = context.WithCancelCause(context.Background())
	return t
}

func RootContext() context.Context {
	return root.ctx
}

func RootContextCanceled() <-chan struct{} {
	return root.ctx.Done()
}

func OnProgramExit(about string, fn func()) {
	root.OnFinished(about, fn)
}

// GracefulShutdown waits for all tasks to finish, up to the given timeout.
//
// If the timeout is exceeded, it prints a list of all tasks that were
// still running when the timeout was reached, and their current tree
// of subtasks.
func GracefulShutdown(timeout time.Duration) (err error) {
	root.cancel(ErrProgramExiting)

	done := make(chan struct{})
	after := time.After(timeout)

	go func() {
		allTasksWg.Wait()
		close(done)
	}()

	for {
		select {
		case <-done:
			return
		case <-after:
			b, err := json.Marshal(DebugTaskList())
			if err != nil {
				logger.Warn().Err(err).Msg("failed to marshal tasks")
				return context.DeadlineExceeded
			}
			logger.Warn().RawJSON("tasks", b).Msgf("Timeout waiting for these %d tasks to finish", allTasks.Size())
			return context.DeadlineExceeded
		}
	}
}

// DebugTaskList returns list of all tasks.
//
// The returned string is suitable for printing to the console.
func DebugTaskList() []string {
	l := make([]string, 0, allTasks.Size())

	allTasks.RangeAll(func(t *Task) {
		l = append(l, t.name)
	})

	slices.Sort(l)
	return l
}
