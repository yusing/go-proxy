package provider

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/rs/zerolog"
	E "github.com/yusing/go-proxy/internal/error"
	R "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/task"
	W "github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type (
	Provider struct {
		ProviderImpl `json:"-"`

		name   string
		t      ProviderType
		routes R.Routes

		watcher W.Watcher
	}
	ProviderImpl interface {
		fmt.Stringer
		loadRoutesImpl() (R.Routes, E.Error)
		NewWatcher() W.Watcher
		Logger() *zerolog.Logger
	}
	ProviderType  string
	ProviderStats struct {
		NumRPs     int          `json:"num_reverse_proxies"`
		NumStreams int          `json:"num_streams"`
		Type       ProviderType `json:"type"`
	}
)

const (
	ProviderTypeDocker ProviderType = "docker"
	ProviderTypeFile   ProviderType = "file"

	providerEventFlushInterval = 300 * time.Millisecond
)

var ErrEmptyProviderName = errors.New("empty provider name")

func newProvider(name string, t ProviderType) *Provider {
	return &Provider{
		name:   name,
		t:      t,
		routes: R.NewRoutes(),
	}
}

func NewFileProvider(filename string) (p *Provider, err error) {
	name := path.Base(filename)
	if name == "" {
		return nil, ErrEmptyProviderName
	}
	p = newProvider(name, ProviderTypeFile)
	p.ProviderImpl, err = FileProviderImpl(filename)
	if err != nil {
		return nil, err
	}
	p.watcher = p.NewWatcher()
	return
}

func NewDockerProvider(name string, dockerHost string) (p *Provider, err error) {
	if name == "" {
		return nil, ErrEmptyProviderName
	}

	p = newProvider(name, ProviderTypeDocker)
	p.ProviderImpl, err = DockerProviderImpl(name, dockerHost, p.IsExplicitOnly())
	if err != nil {
		return nil, err
	}
	p.watcher = p.NewWatcher()
	return
}

func (p *Provider) IsExplicitOnly() bool {
	return p.name[len(p.name)-1] == '!'
}

func (p *Provider) GetName() string {
	return p.name
}

func (p *Provider) GetType() ProviderType {
	return p.t
}

// to work with json marshaller.
func (p *Provider) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Provider) startRoute(parent task.Task, r *R.Route) E.Error {
	subtask := parent.Subtask(p.String() + "/" + r.Entry.Alias)
	err := r.Start(subtask)
	if err != nil {
		p.routes.Delete(r.Entry.Alias)
		subtask.Finish(err) // just to ensure
		return err.Subject(r.Entry.Alias)
	}
	p.routes.Store(r.Entry.Alias, r)
	subtask.OnFinished("del from provider", func() {
		p.routes.Delete(r.Entry.Alias)
	})
	return nil
}

// Start implements task.TaskStarter.
func (p *Provider) Start(configSubtask task.Task) E.Error {
	// routes and event queue will stop on parent cancel
	providerTask := configSubtask

	errs := p.routes.CollectErrorsParallel(
		func(alias string, r *R.Route) error {
			return p.startRoute(providerTask, r)
		})

	eventQueue := events.NewEventQueue(
		providerTask,
		providerEventFlushInterval,
		func(flushTask task.Task, events []events.Event) {
			handler := p.newEventHandler()
			// routes' lifetime should follow the provider's lifetime
			handler.Handle(providerTask, events)
			handler.Log()
			flushTask.Finish("events flushed")
		},
		func(err E.Error) {
			E.LogError("event error", err, p.Logger())
		},
	)
	eventQueue.Start(p.watcher.Events(providerTask.Context()))

	if err := E.Join(errs...); err != nil {
		return err.Subject(p.String())
	}
	return nil
}

func (p *Provider) RangeRoutes(do func(string, *R.Route)) {
	p.routes.RangeAll(do)
}

func (p *Provider) GetRoute(alias string) (*R.Route, bool) {
	return p.routes.Load(alias)
}

func (p *Provider) LoadRoutes() E.Error {
	var err E.Error
	p.routes, err = p.loadRoutesImpl()
	if p.routes.Size() > 0 {
		return err
	}
	if err == nil {
		return nil
	}
	return err
}

func (p *Provider) NumRoutes() int {
	return p.routes.Size()
}

func (p *Provider) Statistics() ProviderStats {
	numRPs := 0
	numStreams := 0
	p.routes.RangeAll(func(_ string, r *R.Route) {
		switch r.Type {
		case R.RouteTypeReverseProxy:
			numRPs++
		case R.RouteTypeStream:
			numStreams++
		}
	})
	return ProviderStats{
		NumRPs:     numRPs,
		NumStreams: numStreams,
		Type:       p.t,
	}
}
