package task

import "context"

type dummyTask struct{}

func DummyTask() (_ Task) {
	return
}

// Context implements Task.
func (d dummyTask) Context() context.Context {
	panic("call of dummyTask.Context")
}

// Finish implements Task.
func (d dummyTask) Finish() {}

// Name implements Task.
func (d dummyTask) Name() string {
	return "Dummy Task"
}

// OnComplete implements Task.
func (d dummyTask) OnComplete(about string, fn func()) {
	panic("call of dummyTask.OnComplete")
}

// Parent implements Task.
func (d dummyTask) Parent() Task {
	panic("call of dummyTask.Parent")
}

// Subtask implements Task.
func (d dummyTask) Subtask(usageFmt string, args ...any) Task {
	panic("call of dummyTask.Subtask")
}

// Wait implements Task.
func (d dummyTask) Wait() {}

// WaitSubTasks implements Task.
func (d dummyTask) WaitSubTasks() {}
