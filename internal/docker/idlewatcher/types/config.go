package types

import (
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

func ValidateConfig(cont *docker.Container) (cfg *Config, res E.Error) {
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

	b := E.NewBuilder("invalid idlewatcher config")
	defer b.To(&res)

	idleTimeout, err := validateDurationPostitive(cont.IdleTimeout)
	b.Add(err.Subjectf("%s", "idle_timeout"))

	wakeTimeout, err := validateDurationPostitive(cont.WakeTimeout)
	b.Add(err.Subjectf("%s", "wake_timeout"))

	stopTimeout, err := validateDurationPostitive(cont.StopTimeout)
	b.Add(err.Subjectf("%s", "stop_timeout"))

	stopMethod, err := validateStopMethod(cont.StopMethod)
	b.Add(err)

	signal, err := validateSignal(cont.StopSignal)
	b.Add(err)

	if err := b.Build(); err != nil {
		return
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

func validateDurationPostitive(value string) (time.Duration, E.Error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, E.Invalid("duration", value).With(err)
	}
	if d < 0 {
		return 0, E.Invalid("duration", "negative value")
	}
	return d, nil
}

func validateSignal(s string) (Signal, E.Error) {
	switch s {
	case "", "SIGINT", "SIGTERM", "SIGHUP", "SIGQUIT",
		"INT", "TERM", "HUP", "QUIT":
		return Signal(s), nil
	}

	return "", E.Invalid("signal", s)
}

func validateStopMethod(s string) (StopMethod, E.Error) {
	sm := StopMethod(s)
	switch sm {
	case StopMethodPause, StopMethodStop, StopMethodKill:
		return sm, nil
	default:
		return "", E.Invalid("stop_method", sm)
	}
}
