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
	Action    uint16
	EventType string
)

const (
	ActionFileModified Action = (1 << iota)
	ActionFileCreated
	ActionFileDeleted

	ActionContainerCreate
	ActionContainerStart
	ActionContainerUnpause

	ActionContainerKill
	ActionContainerStop
	ActionContainerPause
	ActionContainerDie

	actionContainerWakeMask  = ActionContainerCreate | ActionContainerStart | ActionContainerUnpause
	actionContainerSleepMask = ActionContainerKill | ActionContainerStop | ActionContainerPause | ActionContainerDie
)

const (
	EventTypeDocker EventType = "docker"
	EventTypeFile   EventType = "file"
)

var DockerEventMap = map[dockerEvents.Action]Action{
	dockerEvents.ActionCreate:  ActionContainerCreate,
	dockerEvents.ActionStart:   ActionContainerStart,
	dockerEvents.ActionUnPause: ActionContainerUnpause,

	dockerEvents.ActionKill:  ActionContainerKill,
	dockerEvents.ActionStop:  ActionContainerStop,
	dockerEvents.ActionPause: ActionContainerPause,
	dockerEvents.ActionDie:   ActionContainerDie,
}

var dockerActionNameMap = func() (m map[Action]string) {
	m = make(map[Action]string, len(DockerEventMap))
	for k, v := range DockerEventMap {
		m[v] = string(k)
	}
	return
}()

func (e Event) String() string {
	return fmt.Sprintf("%s %s", e.ActorName, e.Action)
}

func (a Action) String() string {
	return dockerActionNameMap[a]
}

func (a Action) IsContainerWake() bool {
	return a&actionContainerWakeMask != 0
}

func (a Action) IsContainerSleep() bool {
	return a&actionContainerSleepMask != 0
}
