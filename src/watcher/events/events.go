package events

import (
	"fmt"

	dockerEvents "github.com/docker/docker/api/types/events"
)

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
	ActionFileModified Action = "modified"
	ActionFileCreated  Action = "created"
	ActionFileDeleted  Action = "deleted"

	ActionDockerStartUnpause Action = "start"
	ActionDockerStopPause    Action = "stop"

	EventTypeDocker EventType = "docker"
	EventTypeFile   EventType = "file"
)

var DockerEventMap = map[dockerEvents.Action]Action{
	dockerEvents.ActionCreate:  ActionDockerStartUnpause,
	dockerEvents.ActionStart:   ActionDockerStartUnpause,
	dockerEvents.ActionPause:   ActionDockerStartUnpause,
	dockerEvents.ActionDie:     ActionDockerStopPause,
	dockerEvents.ActionStop:    ActionDockerStopPause,
	dockerEvents.ActionUnPause: ActionDockerStopPause,
	dockerEvents.ActionKill:    ActionDockerStopPause,
}

func (e Event) String() string {
	return fmt.Sprintf("%s %s", e.ActorName, e.Action)
}

func (a Action) IsDelete() bool {
	return a == ActionFileDeleted
}
