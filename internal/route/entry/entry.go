package entry

import (
	E "github.com/yusing/go-proxy/internal/error"
	route "github.com/yusing/go-proxy/internal/route/types"
)

type Entry = route.Entry

func ValidateEntry(m *route.RawEntry) (Entry, E.Error) {
	scheme, err := route.NewScheme(m.Scheme)
	if err != nil {
		return nil, E.From(err)
	}

	var entry Entry
	errs := E.NewBuilder("entry validation failed")
	if scheme.IsStream() {
		entry = validateStreamEntry(m, errs)
	} else {
		entry = validateRPEntry(m, scheme, errs)
	}
	if errs.HasError() {
		return nil, errs.Error()
	}
	if !UseHealthCheck(entry) && (UseLoadBalance(entry) || UseIdleWatcher(entry)) {
		return nil, E.New("healthCheck.disable cannot be true when loadbalancer or idlewatcher is enabled")
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
