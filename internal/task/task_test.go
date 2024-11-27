package task

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestTaskCreation(t *testing.T) {
	rootTask := GlobalTask("root-task")
	subTask := rootTask.Subtask("subtask")

	ExpectEqual(t, "root-task", rootTask.Name())
	ExpectEqual(t, "subtask", subTask.Name())
}

func TestTaskCancellation(t *testing.T) {
	subTaskDone := make(chan struct{})

	rootTask := GlobalTask("root-task")
	subTask := rootTask.Subtask("subtask")

	go func() {
		subTask.Wait()
		close(subTaskDone)
	}()

	go rootTask.Finish("done")

	select {
	case <-subTaskDone:
		err := subTask.Context().Err()
		ExpectError(t, context.Canceled, err)
		cause := context.Cause(subTask.Context())
		ExpectError(t, ErrTaskCanceled, cause)
	case <-time.After(1 * time.Second):
		t.Fatal("subTask context was not canceled as expected")
	}
}

func TestOnComplete(t *testing.T) {
	rootTask := GlobalTask("root-task")
	task := rootTask.Subtask("test")

	var value atomic.Int32
	task.OnFinished("set value", func() {
		value.Store(1234)
	})
	task.Finish("done")
	ExpectEqual(t, value.Load(), 1234)
}

func TestGlobalContextWait(t *testing.T) {
	testResetGlobalTask()
	defer CancelGlobalContext()

	rootTask := GlobalTask("root-task")

	finished1, finished2 := false, false

	subTask1 := rootTask.Subtask("subtask1")
	subTask2 := rootTask.Subtask("subtask2")
	subTask1.OnFinished("set finished", func() {
		finished1 = true
	})
	subTask2.OnFinished("set finished", func() {
		finished2 = true
	})

	go func() {
		time.Sleep(500 * time.Millisecond)
		subTask1.Finish("done")
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		subTask2.Finish("done")
	}()

	go func() {
		subTask1.Wait()
		subTask2.Wait()
		rootTask.Finish("done")
	}()

	GlobalContextWait(1 * time.Second)
	ExpectTrue(t, finished1)
	ExpectTrue(t, finished2)
	ExpectError(t, context.Canceled, rootTask.Context().Err())
	ExpectError(t, ErrTaskCanceled, context.Cause(subTask1.Context()))
	ExpectError(t, ErrTaskCanceled, context.Cause(subTask2.Context()))
}

func TestTimeoutOnGlobalContextWait(t *testing.T) {
	testResetGlobalTask()

	rootTask := GlobalTask("root-task")
	rootTask.Subtask("subtask")

	ExpectError(t, context.DeadlineExceeded, GlobalContextWait(200*time.Millisecond))
}

func TestGlobalContextCancellation(t *testing.T) {
	testResetGlobalTask()

	taskDone := make(chan struct{})
	rootTask := GlobalTask("root-task")

	go func() {
		rootTask.Wait()
		close(taskDone)
	}()

	CancelGlobalContext()

	select {
	case <-taskDone:
		err := rootTask.Context().Err()
		ExpectError(t, context.Canceled, err)
		cause := context.Cause(rootTask.Context())
		ExpectError(t, ErrProgramExiting, cause)
	case <-time.After(1 * time.Second):
		t.Fatal("subTask context was not canceled as expected")
	}
}
