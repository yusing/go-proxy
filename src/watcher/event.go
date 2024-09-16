package watcher

import "fmt"

type (
	Event struct {
		ActorName string
		Action    Action
	}
	Action string
)

const (
	ActionModified Action = "MODIFIED"
	ActionCreated  Action = "CREATED"
	ActionStarted  Action = "STARTED"
	ActionDeleted  Action = "DELETED"
)

func (e Event) String() string {
	return fmt.Sprintf("%s %s", e.ActorName, e.Action)
}

func (a Action) IsDelete() bool {
	return a == ActionDeleted
}
