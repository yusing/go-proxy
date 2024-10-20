package entry

import (
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/net/http/loadbalancer"
	net "github.com/yusing/go-proxy/internal/net/types"
	T "github.com/yusing/go-proxy/internal/proxy/fields"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type Entry interface {
	TargetName() string
	TargetURL() net.URL
	RawEntry() *RawEntry
	LoadBalanceConfig() *loadbalancer.Config
	HealthCheckConfig() *health.HealthCheckConfig
	IdlewatcherConfig() *idlewatcher.Config
}

func ValidateEntry(m *RawEntry) (Entry, E.Error) {
	m.FillMissingFields()

	scheme, err := T.NewScheme(m.Scheme)
	if err != nil {
		return nil, err
	}

	var entry Entry
	e := E.NewBuilder("error validating entry")
	if scheme.IsStream() {
		entry = validateStreamEntry(m, e)
	} else {
		entry = validateRPEntry(m, scheme, e)
	}
	if err := e.Build(); err != nil {
		return nil, err
	}
	return entry, nil
}

func IsDocker(entry Entry) bool {
	iw := entry.IdlewatcherConfig()
	return iw != nil && iw.ContainerID != ""
}

func IsZeroPort(entry Entry) bool {
	return entry.TargetURL().Port() == "0"
}

func ShouldNotServe(entry Entry) bool {
	return IsZeroPort(entry) && !UseIdleWatcher(entry)
}

func UseLoadBalance(entry Entry) bool {
	lb := entry.LoadBalanceConfig()
	return lb != nil && lb.Link != ""
}

func UseIdleWatcher(entry Entry) bool {
	iw := entry.IdlewatcherConfig()
	return iw != nil && iw.IdleTimeout > 0
}

func UseHealthCheck(entry Entry) bool {
	hc := entry.HealthCheckConfig()
	return hc != nil && !hc.Disable
}
