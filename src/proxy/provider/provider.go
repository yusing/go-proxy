package provider

import (
	"context"
	"fmt"
	"path"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/error"
	R "github.com/yusing/go-proxy/route"
	W "github.com/yusing/go-proxy/watcher"
)

type (
	Provider struct {
		ProviderImpl

		name   string
		t      ProviderType
		routes R.Routes

		watcher       W.Watcher
		watcherCtx    context.Context
		watcherCancel context.CancelFunc

		l *logrus.Entry
	}
	ProviderImpl interface {
		NewWatcher() W.Watcher
		// even returns error, routes must be non-nil
		LoadRoutesImpl() (R.Routes, E.NestedError)
		OnEvent(event W.Event, routes R.Routes) EventResult
	}
	ProviderType string
	EventResult  struct {
		nRemoved int
		nAdded   int
		err      E.NestedError
	}
)

const (
	ProviderTypeDocker ProviderType = "docker"
	ProviderTypeFile   ProviderType = "file"
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
	p = newProvider(name, ProviderTypeFile)
	p.ProviderImpl, err = FileProviderImpl(filename)
	if err != nil {
		return nil, err
	}
	p.watcher = p.NewWatcher()
	return
}

func NewDockerProvider(name string, dockerHost string) (p *Provider, err E.NestedError) {
	p = newProvider(name, ProviderTypeDocker)
	p.ProviderImpl, err = DockerProviderImpl(dockerHost)
	if err != nil {
		return nil, err
	}
	p.watcher = p.NewWatcher()
	return
}

func (p *Provider) GetName() string {
	return p.name
}

func (p *Provider) GetType() ProviderType {
	return p.t
}

func (p *Provider) String() string {
	return fmt.Sprintf("%s-%s", p.t, p.name)
}

func (p *Provider) StartAllRoutes() (res E.NestedError) {
	errors := E.NewBuilder("errors in routes")
	defer errors.To(&res)

	// start watcher no matter load success or not
	p.watcherCtx, p.watcherCancel = context.WithCancel(context.Background())
	go p.watchEvents()

	nStarted := 0
	nFailed := 0

	p.routes.RangeAll(func(alias string, r R.Route) {
		if err := r.Start(); err.HasError() {
			errors.Add(err.Subject(r))
			nFailed++
		} else {
			nStarted++
		}
	})

	p.l.Debugf("%d routes started, %d failed", nStarted, nFailed)
	return
}

func (p *Provider) StopAllRoutes() (res E.NestedError) {
	if p.watcherCancel != nil {
		p.watcherCancel()
		p.watcherCancel = nil
	}

	errors := E.NewBuilder("errors stopping routes for provider %q", p.name)
	defer errors.To(&res)

	nStopped := 0
	nFailed := 0
	p.routes.RangeAll(func(alias string, r R.Route) {
		if err := r.Stop(); err.HasError() {
			errors.Add(err.Subject(r))
			nFailed++
		} else {
			nStopped++
		}
	})
	p.l.Debugf("%d routes stopped, %d failed", nStopped, nFailed)
	return
}

func (p *Provider) RangeRoutes(do func(string, R.Route)) {
	p.routes.RangeAll(do)
}

func (p *Provider) GetRoute(alias string) (R.Route, bool) {
	return p.routes.Load(alias)
}

func (p *Provider) LoadRoutes() E.NestedError {
	routes, err := p.LoadRoutesImpl()
	p.routes = routes
	p.l.Infof("loaded %d routes", routes.Size())
	return err
}

func (p *Provider) watchEvents() {
	events, errs := p.watcher.Events(p.watcherCtx)
	l := p.l.WithField("module", "watcher")

	for {
		select {
		case <-p.watcherCtx.Done():
			return
		case event, ok := <-events:
			if !ok { // channel closed
				return
			}
			res := p.OnEvent(event, p.routes)
			l.Infof("%s event %q", event.Type, event)
			l.Infof("%d route added, %d routes removed", res.nAdded, res.nRemoved)
			if res.err.HasError() {
				l.Error(res.err)
			}
		case err, ok := <-errs:
			if !ok {
				return
			}
			if err.Is(context.Canceled) {
				continue
			}
			l.Errorf("watcher error: %s", err)
		}
	}
}
