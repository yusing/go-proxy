package provider

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/rs/zerolog"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/route/provider/types"
	"github.com/yusing/go-proxy/internal/task"
	W "github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type (
	Provider struct {
		ProviderImpl

		t      types.ProviderType
		routes route.Routes

		watcher W.Watcher
	}
	ProviderImpl interface {
		fmt.Stringer
		ShortName() string
		IsExplicitOnly() bool
		loadRoutesImpl() (route.Routes, E.Error)
		NewWatcher() W.Watcher
		Logger() *zerolog.Logger
	}
)

const (
	providerEventFlushInterval = 300 * time.Millisecond
)

var ErrEmptyProviderName = errors.New("empty provider name")

func newProvider(t types.ProviderType) *Provider {
	return &Provider{t: t}
}

func NewFileProvider(filename string) (p *Provider, err error) {
	name := path.Base(filename)
	if name == "" {
		return nil, ErrEmptyProviderName
	}
	p = newProvider(types.ProviderTypeFile)
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

	p = newProvider(types.ProviderTypeDocker)
	p.ProviderImpl, err = DockerProviderImpl(name, dockerHost)
	if err != nil {
		return nil, err
	}
	p.watcher = p.NewWatcher()
	return
}

func (p *Provider) GetType() types.ProviderType {
	return p.t
}

// to work with json marshaller.
func (p *Provider) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Provider) startRoute(parent task.Parent, r *route.Route) E.Error {
	err := r.Start(parent)
	if err != nil {
		return err.Subject(r.Alias)
	}
	return nil
}

// Start implements task.TaskStarter.
func (p *Provider) Start(parent task.Parent) E.Error {
	t := parent.Subtask("provider."+p.String(), false)

	errs := E.NewBuilder("routes error")
	for alias, r := range p.routes {
		if err := p.startRoute(t, r); err != nil {
			errs.Add(err)
			delete(p.routes, alias)
		}
	}

	eventQueue := events.NewEventQueue(
		t.Subtask("event_queue", false),
		providerEventFlushInterval,
		func(events []events.Event) {
			handler := p.newEventHandler()
			// routes' lifetime should follow the provider's lifetime
			handler.Handle(t, events)
			handler.Log()
		},
		func(err E.Error) {
			E.LogError("event error", err, p.Logger())
		},
	)
	eventQueue.Start(p.watcher.Events(t.Context()))

	if err := errs.Error(); err != nil {
		return err.Subject(p.String())
	}
	return nil
}

func (p *Provider) RangeRoutes(do func(string, *route.Route)) {
	for alias, r := range p.routes {
		do(alias, r)
	}
}

func (p *Provider) GetRoute(alias string) (r *route.Route, ok bool) {
	r, ok = p.routes[alias]
	return
}

func (p *Provider) loadRoutes() (routes route.Routes, err E.Error) {
	routes, err = p.loadRoutesImpl()
	if err != nil && len(routes) == 0 {
		return route.Routes{}, err
	}
	errs := E.NewBuilder("routes error")
	errs.Add(err)
	// check for exclusion
	// set alias and provider, then validate
	for alias, r := range routes {
		r.Alias = alias
		r.Provider = p.ShortName()
		r.Finalize()
		if err := r.Validate(); err != nil {
			errs.Add(err.Subject(alias))
			delete(routes, alias)
			continue
		}
		if r.ShouldExclude() {
			delete(routes, alias)
		}
	}
	return routes, errs.Error()
}

func (p *Provider) LoadRoutes() (err E.Error) {
	p.routes, err = p.loadRoutes()
	return
}

func (p *Provider) NumRoutes() int {
	return len(p.routes)
}
