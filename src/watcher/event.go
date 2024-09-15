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
	ActionDeleted  Action = "DELETED"
	ActionCreated  Action = "CREATED"
)

func (e Event) String() string {
	return fmt.Sprintf("%s %s", e.ActorName, e.Action)
}

func (a Action) IsDelete() bool {
	return a == ActionDeleted
}

func (a Action) IsModify() bool {
	return a == ActionModified
}

func (a Action) IsCreate() bool {
	return a == ActionCreated
}
