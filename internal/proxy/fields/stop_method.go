package fields

import (
	E "github.com/yusing/go-proxy/internal/error"
)

type StopMethod string

const (
	StopMethodPause StopMethod = "pause"
	StopMethodStop  StopMethod = "stop"
	StopMethodKill  StopMethod = "kill"
)

func ValidateStopMethod(s string) (StopMethod, E.NestedError) {
	sm := StopMethod(s)
	switch sm {
	case StopMethodPause, StopMethodStop, StopMethodKill:
		return sm, nil
	default:
		return "", E.Invalid("stop_method", sm)
	}
}
