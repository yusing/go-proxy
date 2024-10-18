package provider

import (
	"fmt"
	"path"
	"time"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/proxy/entry"
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

		l *logrus.Entry
	}
	ProviderImpl interface {
		fmt.Stringer
		NewWatcher() W.Watcher
		LoadRoutesImpl() (R.Routes, E.NestedError)
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

	providerEventFlushInterval = 500 * time.Millisecond
)

func newProvider(name string, t ProviderType) *Provider {
	p := &Provider{
		name:   name,
		t:      t,
		routes: R.NewRoutes(),
	}
	p.l = logrus.WithField("provider", p)
	return p
}

func NewFileProvider(filename string) (p *Provider, err E.NestedError) {
	name := path.Base(filename)
	if name == "" {
		return nil, E.Invalid("file name", "empty")
	}
	p = newProvider(name, ProviderTypeFile)
	p.ProviderImpl, err = FileProviderImpl(filename)
	if err != nil {
		return nil, err
	}
	p.watcher = p.NewWatcher()
	return
}

func NewDockerProvider(name string, dockerHost string) (p *Provider, err E.NestedError) {
	if name == "" {
		return nil, E.Invalid("provider name", "empty")
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

func (p *Provider) startRoute(parent task.Task, r *R.Route) E.NestedError {
	if entry.UseLoadBalance(r) {
		r.Entry.Alias = p.String() + "/" + r.Entry.Alias
	}
	subtask := parent.Subtask(r.Entry.Alias)
	err := r.Start(subtask)
	if err != nil {
		p.routes.Delete(r.Entry.Alias)
		subtask.Finish(err.String()) // just to ensure
		return err
	} else {
		subtask.OnComplete("del from provider", func() {
			p.routes.Delete(r.Entry.Alias)
		})
	}
	return nil
}

// Start implements task.TaskStarter.
func (p *Provider) Start(configSubtask task.Task) (res E.NestedError) {
	errors := E.NewBuilder("errors starting routes")
	defer errors.To(&res)

	// routes and event queue will stop on parent cancel
	providerTask := configSubtask

	p.routes.RangeAllParallel(func(alias string, r *R.Route) {
		errors.Add(p.startRoute(providerTask, r))
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
		func(err E.NestedError) {
			p.l.Error(err)
		},
	)
	eventQueue.Start(p.watcher.Events(providerTask.Context()))
	return
}

func (p *Provider) RangeRoutes(do func(string, *R.Route)) {
	p.routes.RangeAll(do)
}

func (p *Provider) GetRoute(alias string) (*R.Route, bool) {
	return p.routes.Load(alias)
}

func (p *Provider) LoadRoutes() E.NestedError {
	var err E.NestedError
	p.routes, err = p.LoadRoutesImpl()
	if p.routes.Size() > 0 {
		return err
	}
	if err == nil {
		return nil
	}
	return E.FailWith("loading routes", err)
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
