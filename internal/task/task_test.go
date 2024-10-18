package task_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/yusing/go-proxy/internal/task"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestTaskCreation(t *testing.T) {
	defer CancelGlobalContext()

	rootTask := GlobalTask("root-task")
	subTask := rootTask.Subtask("subtask")

	ExpectEqual(t, "root-task", rootTask.Name())
	ExpectEqual(t, "subtask", subTask.Name())
}

func TestTaskCancellation(t *testing.T) {
	defer CancelGlobalContext()

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
		ExpectError(t, ErrTaskCancelled, cause)
	case <-time.After(1 * time.Second):
		t.Fatal("subTask context was not canceled as expected")
	}
}

func TestGlobalContextCancellation(t *testing.T) {
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

func TestOnComplete(t *testing.T) {
	defer CancelGlobalContext()

	task := GlobalTask("test")
	var value atomic.Int32
	task.OnComplete("set value", func() {
		value.Store(1234)
	})
	task.Finish("done")
	ExpectEqual(t, value.Load(), 1234)
}

func TestGlobalContextWait(t *testing.T) {
	defer CancelGlobalContext()

	rootTask := GlobalTask("root-task")

	finished1, finished2 := false, false

	subTask1 := rootTask.Subtask("subtask1")
	subTask2 := rootTask.Subtask("subtask2")
	subTask1.OnComplete("set finished", func() {
		finished1 = true
	})
	subTask2.OnComplete("set finished", func() {
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
	ExpectError(t, ErrTaskCancelled, context.Cause(subTask1.Context()))
	ExpectError(t, ErrTaskCancelled, context.Cause(subTask2.Context()))
}

func TestTimeoutOnGlobalContextWait(t *testing.T) {
	defer CancelGlobalContext()

	rootTask := GlobalTask("root-task")
	subTask := rootTask.Subtask("subtask")

	done := make(chan struct{})
	go func() {
		GlobalContextWait(500 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("GlobalContextWait should have timed out")
	case <-time.After(200 * time.Millisecond):
	}

	// Ensure clean exit
	subTask.Finish("exit")
}

func TestGlobalContextCancel(t *testing.T) {
}
