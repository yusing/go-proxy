package entry

import (
	"fmt"

	"github.com/yusing/go-proxy/internal/docker"
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	net "github.com/yusing/go-proxy/internal/net/types"
	route "github.com/yusing/go-proxy/internal/route/types"
)

type StreamEntry struct {
	Raw *route.RawEntry `json:"raw"`

	Scheme    route.StreamScheme `json:"scheme"`
	URL       net.URL            `json:"url"`
	ListenURL net.URL            `json:"listening_url"`
	Port      route.StreamPort   `json:"port,omitempty"`

	/* Docker only */
	Idlewatcher *idlewatcher.Config `json:"idlewatcher,omitempty"`
}

func (s *StreamEntry) TargetName() string {
	return s.Raw.Alias
}

func (s *StreamEntry) TargetURL() net.URL {
	return s.URL
}

func (s *StreamEntry) RawEntry() *route.RawEntry {
	return s.Raw
}

func (s *StreamEntry) IdlewatcherConfig() *idlewatcher.Config {
	return s.Idlewatcher
}

func validateStreamEntry(m *route.RawEntry, errs *E.Builder) *StreamEntry {
	cont := m.Container
	if cont == nil {
		cont = docker.DummyContainer
	}

	port := E.Collect(errs, route.ValidateStreamPort, m.Port)
	scheme := E.Collect(errs, route.ValidateStreamScheme, m.Scheme)
	url := E.Collect(errs, net.ParseURL, fmt.Sprintf("%s://%s:%d", scheme.ProxyScheme, m.Host, port.ProxyPort))
	listenURL := E.Collect(errs, net.ParseURL, fmt.Sprintf("%s://:%d", scheme.ListeningScheme, port.ListeningPort))
	idleWatcherCfg := E.Collect(errs, idlewatcher.ValidateConfig, cont)

	if errs.HasError() {
		return nil
	}

	return &StreamEntry{
		Raw:         m,
		Scheme:      *scheme,
		URL:         url,
		ListenURL:   listenURL,
		Port:        port,
		Idlewatcher: idleWatcherCfg,
	}
}
