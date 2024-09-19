package event

import "fmt"

type (
	Event struct {
		Type            EventType
		ActorName       string
		ActorID         string
		ActorAttributes map[string]string
		Action          Action
	}
	Action    string
	EventType string
)

const (
	ActionModified Action = "modified"
	ActionCreated  Action = "created"
	ActionStarted  Action = "started"
	ActionDeleted  Action = "deleted"
	ActionStopped  Action = "stopped"

	EventTypeDocker EventType = "docker"
	EventTypeFile   EventType = "file"
)

func (e Event) String() string {
	return fmt.Sprintf("%s %s", e.ActorName, e.Action)
}

func (a Action) IsDelete() bool {
	return a == ActionDeleted
}
