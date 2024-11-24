package idlewatcher

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/rs/zerolog"
	D "github.com/yusing/go-proxy/internal/docker"
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	route "github.com/yusing/go-proxy/internal/route/types"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	F "github.com/yusing/go-proxy/internal/utils/functional"
	"github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type (
	Watcher struct {
		_ U.NoCopy

		zerolog.Logger

		*idlewatcher.Config
		*waker

		client       D.Client
		stopByMethod StopCallback // send a docker command w.r.t. `stop_method`
		ticker       *time.Ticker
		task         task.Task
	}

	WakeDone     <-chan error
	WakeFunc     func() WakeDone
	StopCallback func() error
)

var (
	watcherMap   = F.NewMapOf[string, *Watcher]()
	watcherMapMu sync.Mutex

	logger = logging.With().Str("module", "idle_watcher").Logger()
)

const dockerReqTimeout = 3 * time.Second

func registerWatcher(providerSubtask task.Task, entry route.Entry, waker *waker) (*Watcher, error) {
	cfg := entry.IdlewatcherConfig()

	if cfg.IdleTimeout == 0 {
		panic("should not reach here")
	}

	watcherMapMu.Lock()
	defer watcherMapMu.Unlock()

	key := cfg.ContainerID

	if w, ok := watcherMap.Load(key); ok {
		w.Config = cfg
		w.waker = waker
		w.resetIdleTimer()
		providerSubtask.Finish("used existing watcher")
		return w, nil
	}

	client, err := D.ConnectClient(cfg.DockerHost)
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		Logger: logger.With().Str("name", cfg.ContainerName).Logger(),
		Config: cfg,
		waker:  waker,
		client: client,
		task:   providerSubtask,
		ticker: time.NewTicker(cfg.IdleTimeout),
	}
	w.stopByMethod = w.getStopCallback()
	watcherMap.Store(key, w)

	go func() {
		cause := w.watchUntilDestroy()

		watcherMap.Delete(w.ContainerID)

		w.ticker.Stop()
		w.client.Close()
		w.task.Finish(cause)
	}()

	return w, nil
}

func (w *Watcher) Wake() error {
	return w.wakeIfStopped()
}

// WakeDebug logs a debug message related to waking the container.
func (w *Watcher) WakeDebug() *zerolog.Event {
	return w.Debug().Str("action", "wake")
}

func (w *Watcher) WakeTrace() *zerolog.Event {
	return w.Trace().Str("action", "wake")
}

func (w *Watcher) WakeError(err error) {
	w.Err(err).Str("action", "wake").Msg("error")
}

func (w *Watcher) LogReason(action, reason string) {
	w.Info().Str("reason", reason).Msg(action)
}

func (w *Watcher) containerStop(ctx context.Context) error {
	return w.client.ContainerStop(ctx, w.ContainerID, container.StopOptions{
		Signal:  string(w.StopSignal),
		Timeout: &w.StopTimeout,
	})
}

func (w *Watcher) containerPause(ctx context.Context) error {
	return w.client.ContainerPause(ctx, w.ContainerID)
}

func (w *Watcher) containerKill(ctx context.Context) error {
	return w.client.ContainerKill(ctx, w.ContainerID, string(w.StopSignal))
}

func (w *Watcher) containerUnpause(ctx context.Context) error {
	return w.client.ContainerUnpause(ctx, w.ContainerID)
}

func (w *Watcher) containerStart(ctx context.Context) error {
	return w.client.ContainerStart(ctx, w.ContainerID, container.StartOptions{})
}

func (w *Watcher) containerStatus() (string, error) {
	if !w.client.Connected() {
		return "", errors.New("docker client not connected")
	}
	ctx, cancel := context.WithTimeoutCause(w.task.Context(), dockerReqTimeout, errors.New("docker request timeout"))
	defer cancel()
	json, err := w.client.ContainerInspect(ctx, w.ContainerID)
	if err != nil {
		return "", err
	}
	return json.State.Status, nil
}

func (w *Watcher) wakeIfStopped() error {
	if w.ContainerRunning {
		return nil
	}

	status, err := w.containerStatus()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(w.task.Context(), w.WakeTimeout)
	defer cancel()

	// !Hard coded here since theres no constants from Docker API
	switch status {
	case "exited", "dead":
		return w.containerStart(ctx)
	case "paused":
		return w.containerUnpause(ctx)
	case "running":
		return nil
	default:
		panic("should not reach here")
	}
}

func (w *Watcher) getStopCallback() StopCallback {
	var cb func(context.Context) error
	switch w.StopMethod {
	case idlewatcher.StopMethodPause:
		cb = w.containerPause
	case idlewatcher.StopMethodStop:
		cb = w.containerStop
	case idlewatcher.StopMethodKill:
		cb = w.containerKill
	default:
		panic("should not reach here")
	}
	return func() error {
		ctx, cancel := context.WithTimeout(w.task.Context(), time.Duration(w.StopTimeout)*time.Second)
		defer cancel()
		return cb(ctx)
	}
}

func (w *Watcher) resetIdleTimer() {
	w.Trace().Msg("reset idle timer")
	w.ticker.Reset(w.IdleTimeout)
}

func (w *Watcher) getEventCh(dockerWatcher watcher.DockerWatcher) (eventTask task.Task, eventCh <-chan events.Event, errCh <-chan E.Error) {
	eventTask = w.task.Subtask("docker event watcher")
	eventCh, errCh = dockerWatcher.EventsWithOptions(eventTask.Context(), watcher.DockerListOptions{
		Filters: watcher.NewDockerFilter(
			watcher.DockerFilterContainer,
			watcher.DockerFilterContainerNameID(w.ContainerID),
			watcher.DockerFilterStart,
			watcher.DockerFilterStop,
			watcher.DockerFilterDie,
			watcher.DockerFilterKill,
			watcher.DockerFilterDestroy,
			watcher.DockerFilterPause,
			watcher.DockerFilterUnpause,
		),
	})
	return
}

// watchUntilDestroy waits for the container to be created, started, or unpaused,
// and then reset the idle timer.
//
// When the container is stopped, paused,
// or killed, the idle timer is stopped and the ContainerRunning flag is set to false.
//
// When the idle timer fires, the container is stopped according to the
// stop method.
//
// it exits only if the context is canceled, the container is destroyed,
// errors occurred on docker client, or route provider died (mainly caused by config reload).
func (w *Watcher) watchUntilDestroy() (returnCause error) {
	dockerWatcher := watcher.NewDockerWatcherWithClient(w.client)
	eventTask, dockerEventCh, dockerEventErrCh := w.getEventCh(dockerWatcher)
	defer eventTask.Finish("stopped")

	for {
		select {
		case <-w.task.Context().Done():
			return w.task.FinishCause()
		case err := <-dockerEventErrCh:
			if !err.Is(context.Canceled) {
				E.LogError("idlewatcher error", err, &w.Logger)
			}
			return err
		case e := <-dockerEventCh:
			switch {
			case e.Action == events.ActionContainerDestroy:
				w.ContainerRunning = false
				w.ready.Store(false)
				w.LogReason("watcher stopped", "container destroyed")
				return errors.New("container destroyed")
			// create / start / unpause
			case e.Action.IsContainerWake():
				w.ContainerRunning = true
				w.resetIdleTimer()
				w.Info().Msg("awaken")
			case e.Action.IsContainerSleep(): // stop / pause / kil
				w.ContainerRunning = false
				w.ready.Store(false)
				w.ticker.Stop()
			default:
				w.Error().Msg("unexpected docker event: " + e.String())
			}
			// container name changed should also change the container id
			if w.ContainerName != e.ActorName {
				w.Debug().Msgf("renamed %s -> %s", w.ContainerName, e.ActorName)
				w.ContainerName = e.ActorName
			}
			if w.ContainerID != e.ActorID {
				w.Debug().Msgf("id changed %s -> %s", w.ContainerID, e.ActorID)
				w.ContainerID = e.ActorID
				// recreate event stream
				eventTask.Finish("recreate event stream")
				eventTask, dockerEventCh, dockerEventErrCh = w.getEventCh(dockerWatcher)
			}
		case <-w.ticker.C:
			w.ticker.Stop()
			if w.ContainerRunning {
				err := w.stopByMethod()
				switch {
				case errors.Is(err, context.Canceled):
					continue
				case err != nil:
					w.Err(err).Msgf("container stop with method %q failed", w.StopMethod)
				default:
					w.LogReason("container stopped", "idle timeout")
				}
			}
		}
	}
}
