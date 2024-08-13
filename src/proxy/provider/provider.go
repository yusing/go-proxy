package provider

import (
	"context"
	"fmt"
	"path"

	"github.com/sirupsen/logrus"
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
	t           ProviderType
	routes      *R.Routes
	reloadReqCh chan struct{}

	watcher       W.Watcher
	watcherCtx    context.Context
	watcherCancel context.CancelFunc

	l *logrus.Entry
}

type ProviderType string

const (
	ProviderTypeDocker ProviderType = "docker"
	ProviderTypeFile   ProviderType = "file"
)

func newProvider(name string, t ProviderType) *Provider {
	return &Provider{
		name:        name,
		t:           t,
		routes:      R.NewRoutes(),
		reloadReqCh: make(chan struct{}, 1),
		l:           logrus.WithField("provider", name),
	}
}
func NewFileProvider(filename string) *Provider {
	name := path.Base(filename)
	p := newProvider(name, ProviderTypeFile)
	p.ProviderImpl = FileProviderImpl(filename)
	p.watcher = p.NewWatcher()
	return p
}

func NewDockerProvider(name string, dockerHost string) *Provider {
	p := newProvider(name, ProviderTypeDocker)
	p.ProviderImpl = DockerProviderImpl(dockerHost)
	p.watcher = p.NewWatcher()
	return p
}

func (p *Provider) GetName() string {
	return p.name
}

func (p *Provider) GetType() ProviderType {
	return p.t
}

func (p *Provider) String() string {
	return fmt.Sprintf("%s (%s provider)", p.name, p.t)
}

func (p *Provider) StartAllRoutes() E.NestedError {
	err := p.loadRoutes()

	// start watcher no matter load success or not
	p.watcherCtx, p.watcherCancel = context.WithCancel(context.Background())
	go p.watchEvents()

	errors := E.NewBuilder("errors in routes")
	nStarted := 0
	nFailed := 0

	if err.IsNotNil() {
		errors.Add(err)
	}

	p.routes.EachKVParallel(func(alias string, r R.Route) {
		if err := r.Start(); err.IsNotNil() {
			errors.Add(err.Subject(r))
			nFailed++
		} else {
			nStarted++
		}
	})
	p.l.Infof("%d routes started, %d failed", nStarted, nFailed)
	return errors.Build()
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
			errors.Add(err.Subject(r))
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
		p.l.Warn(err.Subject(p))
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
