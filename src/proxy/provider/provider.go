package provider

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	R "github.com/yusing/go-proxy/route"
	W "github.com/yusing/go-proxy/watcher"
)

type ProviderImpl interface {
	GetProxyEntries() (M.ProxyEntries, E.NestedError)
	NewWatcher() W.Watcher
}

type Provider struct {
	ProviderImpl

	name        string
	routes      *R.Routes
	reloadReqCh chan struct{}

	watcher       W.Watcher
	watcherCtx    context.Context
	watcherCancel context.CancelFunc

	l *logrus.Entry
}

func NewProvider(name string, model M.ProxyProvider) (p *Provider) {
	p = &Provider{
		name:        name,
		routes:      R.NewRoutes(),
		reloadReqCh: make(chan struct{}, 1),
		l:           logrus.WithField("provider", name),
	}
	switch model.Kind {
	case common.ProviderKind_Docker:
		p.ProviderImpl = DockerProviderImpl(&model)
	case common.ProviderKind_File:
		p.ProviderImpl = FileProviderImpl(&model)
	}
	p.watcher = p.NewWatcher()
	return
}

func (p *Provider) GetName() string {
	return p.name
}

func (p *Provider) StartAllRoutes() E.NestedError {
	err := p.loadRoutes()

	// start watcher no matter load success or not
	p.watcherCtx, p.watcherCancel = context.WithCancel(context.Background())
	go p.watchEvents()

	if err.IsNotNil() {
		return err
	}
	errors := E.NewBuilder("errors starting routes for provider %q", p.name)
	nStarted := 0
	p.routes.EachKVParallel(func(alias string, r R.Route) {
		if err := r.Start(); err.IsNotNil() {
			errors.Add(err.Subject(alias))
		} else {
			nStarted++
		}
	})
	if err := errors.Build(); err.IsNotNil() {
		return err
	}
	p.l.Infof("%d routes started", nStarted)
	return E.Nil()
}

func (p *Provider) StopAllRoutes() E.NestedError {
	defer p.routes.Clear()

	if p.watcherCancel != nil {
		p.watcherCancel()
	}
	errors := E.NewBuilder("errors stopping routes for provider %q", p.name)
	nStopped := 0
	p.routes.EachKVParallel(func(alias string, r R.Route) {
		if err := r.Stop(); err.IsNotNil() {
			errors.Add(err.Subject(alias))
		} else {
			nStopped++
		}
	})
	if err := errors.Build(); err.IsNotNil() {
		return err
	}
	p.l.Infof("%d routes stopped", nStopped)
	return E.Nil()
}

func (p *Provider) ReloadRoutes() {
	defer p.l.Info("routes reloaded")

	select {
	case p.reloadReqCh <- struct{}{}:
		defer func() {
			<-p.reloadReqCh
		}()
		p.StopAllRoutes()
		p.loadRoutes()
		p.StartAllRoutes()
	default:
		return
	}
}

func (p *Provider) GetCurrentRoutes() *R.Routes {
	return p.routes
}

func (p *Provider) watchEvents() {
	events, errs := p.watcher.Events(p.watcherCtx)
	l := logrus.WithField("?", "watcher")

	for {
		select {
		case <-p.reloadReqCh:
			p.ReloadRoutes()
		case event, ok := <-events:
			if !ok {
				return
			}
			l.Infof("watcher event: %v", event)
			p.reloadReqCh <- struct{}{}
		case err, ok := <-errs:
			if !ok {
				return
			}
			l.Errorf("watcher error: %s", err)
		}
	}
}

func (p *Provider) loadRoutes() E.NestedError {
	entries, err := p.GetProxyEntries()

	if err.IsNotNil() {
		p.l.Warn(err.Subjectf("provider %s", p.name))
	}
	p.routes = R.NewRoutes()

	errors := E.NewBuilder("errors loading routes from provider %q", p.name)
	entries.EachKV(func(a string, e *M.ProxyEntry) {
		e.Alias = a
		r, err := R.NewRoute(e)
		if err.IsNotNil() {
			errors.Add(err.Subject(a))
			p.l.Debugf("failed to load route: %s, %s", a, err)
		} else {
			p.routes.Set(a, r)
		}
	})
	p.l.Debugf("loaded %d routes from %d entries", p.routes.Size(), entries.Size())
	return errors.Build()
}
