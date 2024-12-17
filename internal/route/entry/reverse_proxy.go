package entry

import (
	"fmt"
	"net/url"

	"github.com/yusing/go-proxy/internal/docker"
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	net "github.com/yusing/go-proxy/internal/net/types"
	route "github.com/yusing/go-proxy/internal/route/types"
)

type ReverseProxyEntry struct { // real model after validation
	Raw *route.RawEntry `json:"raw"`
	URL net.URL         `json:"url"`

	/* Docker only */
	Idlewatcher *idlewatcher.Config `json:"idlewatcher,omitempty"`
}

func (rp *ReverseProxyEntry) TargetName() string {
	return rp.Raw.Alias
}

func (rp *ReverseProxyEntry) TargetURL() net.URL {
	return rp.URL
}

func (rp *ReverseProxyEntry) RawEntry() *route.RawEntry {
	return rp.Raw
}

func (rp *ReverseProxyEntry) IdlewatcherConfig() *idlewatcher.Config {
	return rp.Idlewatcher
}

func validateRPEntry(m *route.RawEntry, s route.Scheme, errs *E.Builder) *ReverseProxyEntry {
	cont := m.Container
	if cont == nil {
		cont = docker.DummyContainer
	}

	if m.LoadBalance != nil && m.LoadBalance.Link == "" {
		m.LoadBalance = nil
	}

	port := E.Collect(errs, route.ValidatePort, m.Port)
	url := E.Collect(errs, url.Parse, fmt.Sprintf("%s://%s:%d", s, m.Host, port))
	iwCfg := E.Collect(errs, idlewatcher.ValidateConfig, cont)

	if errs.HasError() {
		return nil
	}

	return &ReverseProxyEntry{
		Raw:         m,
		URL:         net.NewURL(url),
		Idlewatcher: iwCfg,
	}
}
