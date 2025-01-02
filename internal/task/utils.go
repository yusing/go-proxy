package task

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/yusing/go-proxy/internal/logging"
	F "github.com/yusing/go-proxy/internal/utils/functional"
)

var ErrProgramExiting = errors.New("program exiting")

var logger = logging.With().Str("module", "task").Logger()

var (
	root     = newRoot()
	allTasks = F.NewSet[*Task]()
)

func testCleanup() {
	root = newRoot()
	allTasks.Clear()
}

// RootTask returns a new Task with the given name, derived from the root context.
func RootTask(name string, needFinish ...bool) *Task {
	return root.Subtask(name, needFinish...)
}

func newRoot() *Task {
	t := &Task{name: "root"}
	t.ctx, t.cancel = context.WithCancelCause(context.Background())
	t.callbacks = make(map[*Callback]struct{})
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
	go root.Finish(ErrProgramExiting)

	after := time.After(timeout)
	for {
		select {
		case <-root.finished:
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
