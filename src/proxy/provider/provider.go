package provider

import (
	"context"
	"fmt"
	"path"
	"time"

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

	cooldownCh chan struct{}
}

type ProviderType string

const (
	ProviderTypeDocker ProviderType = "docker"
	ProviderTypeFile   ProviderType = "file"
)

func newProvider(name string, t ProviderType) *Provider {
	p := &Provider{
		name:        name,
		t:           t,
		routes:      R.NewRoutes(),
		reloadReqCh: make(chan struct{}, 1),
		cooldownCh:  make(chan struct{}, 1),
	}
	p.l = logrus.WithField("provider", p)
	go p.processReloadRequests()
	return p
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
	return fmt.Sprintf("%s: %s", p.t, p.name)
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

	p.l.Debugf("%d routes started, %d failed", nStarted, nFailed)
	return errors.Build()
}

func (p *Provider) StopAllRoutes() E.NestedError {
	if p.watcherCancel != nil {
		p.watcherCancel()
		p.watcherCancel = nil
	}
	errors := E.NewBuilder("errors stopping routes for provider %q", p.name)
	nStopped := 0
	nFailed := 0
	p.routes.EachKVParallel(func(alias string, r R.Route) {
		if err := r.Stop(); err.IsNotNil() {
			errors.Add(err.Subject(r))
			nFailed++
		} else {
			nStopped++
		}
	})
	p.l.Debugf("%d routes stopped, %d failed", nStopped, nFailed)
	return errors.Build()
}

func (p *Provider) ReloadRoutes() {
	select {
	case p.reloadReqCh <- struct{}{}:
		// Successfully sent reload request
	default:
		// Reload request already in progress, ignore this request
	}
}

func (p *Provider) GetCurrentRoutes() *R.Routes {
	return p.routes
}

func (p *Provider) watchEvents() {
	events, errs := p.watcher.Events(p.watcherCtx)
	l := p.l.WithField("module", "watcher")

	for {
		select {
		case <-p.watcherCtx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			l.Info(event)
			p.ReloadRoutes()
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

func (p *Provider) processReloadRequests() {
	for range p.reloadReqCh {
		// prevent busy loop caused by a container
		// repeating crashing and restarting
		select {
		case p.cooldownCh <- struct{}{}:
			p.l.Info("Starting to reload routes")
			nRoutes := p.routes.Size()

			p.StopAllRoutes()
			p.loadRoutes()
			p.StartAllRoutes()

			p.l.Infof("Routes reloaded (%d -> %d)", nRoutes, p.routes.Size())

			go func() {
				time.Sleep(reloadCooldown)
				<-p.cooldownCh
			}()
		default:
		}
	}
}

func (p *Provider) loadRoutes() E.NestedError {
	entries, err := p.GetProxyEntries()

	if err.IsNotNil() {
		p.l.Warn(err.Subject(p))
	}
	p.routes = R.NewRoutes()

	errors := E.NewBuilder("errors loading routes from %s", p)
	entries.EachKV(func(a string, e *M.ProxyEntry) {
		e.Alias = a
		r, err := R.NewRoute(e)
		if err.IsNotNil() {
			errors.Add(err.Subject(a))
		} else {
			p.routes.Set(a, r)
		}
	})
	return errors.Build()
}

const reloadCooldown = 50 * time.Millisecond
