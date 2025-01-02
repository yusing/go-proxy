package task

import "strings"

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
