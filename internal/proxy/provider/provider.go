package provider

import (
	"context"
	"path"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/internal/error"
	R "github.com/yusing/go-proxy/internal/route"
	W "github.com/yusing/go-proxy/internal/watcher"
)

type (
	Provider struct {
		ProviderImpl `json:"-"`

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
		String() string
	}
	ProviderType  string
	ProviderStats struct {
		NumRPs     int          `json:"num_reverse_proxies"`
		NumStreams int          `json:"num_streams"`
		Type       ProviderType `json:"type"`
	}
	EventResult struct {
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
	p.ProviderImpl, err = DockerProviderImpl(dockerHost, p.IsExplicitOnly())
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

// to work with json marshaller
func (p *Provider) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Provider) StartAllRoutes() (res E.NestedError) {
	errors := E.NewBuilder("errors in routes")
	defer errors.To(&res)

	// start watcher no matter load success or not
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
	var err E.NestedError
	p.routes, err = p.LoadRoutesImpl()
	if p.routes.Size() > 0 {
		p.l.Infof("loaded %d routes", p.routes.Size())
		return err
	}
	return E.FailWith("loading routes", err)
}

func (p *Provider) Statistics() ProviderStats {
	numRPs := 0
	numStreams := 0
	p.routes.RangeAll(func(_ string, r R.Route) {
		if !r.Started() {
			return
		}
		switch r.Type() {
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

func (p *Provider) watchEvents() {
	p.watcherCtx, p.watcherCancel = context.WithCancel(context.Background())
	events, errs := p.watcher.Events(p.watcherCtx)
	l := p.l.WithField("module", "watcher")

	for {
		select {
		case <-p.watcherCtx.Done():
			return
		case event := <-events:
			res := p.OnEvent(event, p.routes)
			l.Infof("%s event %q", event.Type, event)
			if res.nAdded > 0 || res.nRemoved > 0 {
				l.Infof("%d route added, %d routes removed", res.nAdded, res.nRemoved)
			}
			if res.err.HasError() {
				l.Error(res.err)
			}
		case err := <-errs:
			if err == nil || err.Is(context.Canceled) {
				continue
			}
			l.Errorf("watcher error: %s", err)
		}
	}
}
