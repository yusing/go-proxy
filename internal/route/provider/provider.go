package provider

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/rs/zerolog"
	E "github.com/yusing/go-proxy/internal/error"
	R "github.com/yusing/go-proxy/internal/route"
	"github.com/yusing/go-proxy/internal/route/provider/types"
	"github.com/yusing/go-proxy/internal/task"
	W "github.com/yusing/go-proxy/internal/watcher"
	"github.com/yusing/go-proxy/internal/watcher/events"
)

type (
	Provider struct {
		ProviderImpl `json:"-"`

		t      types.ProviderType
		routes R.Routes

		watcher W.Watcher
	}
	ProviderImpl interface {
		fmt.Stringer
		ShortName() string
		IsExplicitOnly() bool
		loadRoutesImpl() (R.Routes, E.Error)
		NewWatcher() W.Watcher
		Logger() *zerolog.Logger
	}
)

const (
	providerEventFlushInterval = 300 * time.Millisecond
)

var ErrEmptyProviderName = errors.New("empty provider name")

func newProvider(t types.ProviderType) *Provider {
	return &Provider{
		t:      t,
		routes: R.NewRoutes(),
	}
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

func (p *Provider) startRoute(parent task.Parent, r *R.Route) E.Error {
	err := r.Start(parent)
	if err != nil {
		return err.Subject(r.Entry.Alias)
	}

	p.routes.Store(r.Entry.Alias, r)
	return nil
}

// Start implements task.TaskStarter.
func (p *Provider) Start(parent task.Parent) E.Error {
	t := parent.Subtask("provider."+p.String(), false)

	// routes and event queue will stop on config reload
	errs := p.routes.CollectErrorsParallel(
		func(alias string, r *R.Route) error {
			return p.startRoute(t, r)
		})

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
