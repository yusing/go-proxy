package types

import (
	"errors"
	"time"

	"github.com/yusing/go-proxy/internal/docker"
	E "github.com/yusing/go-proxy/internal/error"
)

type (
	Config struct {
		IdleTimeout time.Duration `json:"idle_timeout,omitempty"`
		WakeTimeout time.Duration `json:"wake_timeout,omitempty"`
		StopTimeout int           `json:"stop_timeout,omitempty"` // docker api takes integer seconds for timeout argument
		StopMethod  StopMethod    `json:"stop_method,omitempty"`
		StopSignal  Signal        `json:"stop_signal,omitempty"`

		DockerHost       string `json:"docker_host,omitempty"`
		ContainerName    string `json:"container_name,omitempty"`
		ContainerID      string `json:"container_id,omitempty"`
		ContainerRunning bool   `json:"container_running,omitempty"`
	}
	StopMethod string
	Signal     string
)

const (
	StopMethodPause StopMethod = "pause"
	StopMethodStop  StopMethod = "stop"
	StopMethodKill  StopMethod = "kill"
)

var validSignals = map[string]struct{}{
	"":       {},
	"SIGINT": {}, "SIGTERM": {}, "SIGHUP": {}, "SIGQUIT": {},
	"INT": {}, "TERM": {}, "HUP": {}, "QUIT": {},
}

func ValidateConfig(cont *docker.Container) (*Config, E.Error) {
	if cont == nil {
		return nil, nil
	}

	if cont.IdleTimeout == "" {
		return &Config{
			DockerHost:       cont.DockerHost,
			ContainerName:    cont.ContainerName,
			ContainerID:      cont.ContainerID,
			ContainerRunning: cont.Running,
		}, nil
	}

	errs := E.NewBuilder("invalid idlewatcher config")

	idleTimeout := E.Collect(errs, validateDurationPostitive, cont.IdleTimeout)
	wakeTimeout := E.Collect(errs, validateDurationPostitive, cont.WakeTimeout)
	stopTimeout := E.Collect(errs, validateDurationPostitive, cont.StopTimeout)
	stopMethod := E.Collect(errs, validateStopMethod, cont.StopMethod)
	signal := E.Collect(errs, validateSignal, cont.StopSignal)

	if errs.HasError() {
		return nil, errs.Error()
	}

	return &Config{
		IdleTimeout: idleTimeout,
		WakeTimeout: wakeTimeout,
		StopTimeout: int(stopTimeout.Seconds()),
		StopMethod:  stopMethod,
		StopSignal:  signal,

		DockerHost:       cont.DockerHost,
		ContainerName:    cont.ContainerName,
		ContainerID:      cont.ContainerID,
		ContainerRunning: cont.Running,
	}, nil
}

func validateDurationPostitive(value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, errors.New("duration must be positive")
	}
	return d, nil
}

func validateSignal(s string) (Signal, error) {
	if _, ok := validSignals[s]; ok {
		return Signal(s), nil
	}
	return "", errors.New("invalid signal " + s)
}

func validateStopMethod(s string) (StopMethod, error) {
	sm := StopMethod(s)
	switch sm {
	case StopMethodPause, StopMethodStop, StopMethodKill:
		return sm, nil
	default:
		return "", errors.New("invalid stop method " + s)
	}
}
