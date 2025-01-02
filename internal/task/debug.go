package task

import (
	"slices"
	"strings"
)

// debug only.
func (t *Task) listChildren() []string {
	var children []string
	allTasks.Range(func(child *Task) bool {
		if child.parent == t {
			children = append(children, strings.TrimPrefix(child.name, t.name+"."))
		}
		return true
	})
	return children
}

// debug only.
func (t *Task) listCallbacks() []string {
	var callbacks []string
	t.mu.Lock()
	defer t.mu.Unlock()
	for c := range t.callbacks {
		callbacks = append(callbacks, c.about)
	}
	return callbacks
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
