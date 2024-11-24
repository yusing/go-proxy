package types

import (
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	loadbalance "github.com/yusing/go-proxy/internal/net/http/loadbalancer/types"
	net "github.com/yusing/go-proxy/internal/net/types"
	"github.com/yusing/go-proxy/internal/watcher/health"
)

type Entry interface {
	TargetName() string
	TargetURL() net.URL
	RawEntry() *RawEntry
	LoadBalanceConfig() *loadbalance.Config
	HealthCheckConfig() *health.HealthCheckConfig
	IdlewatcherConfig() *idlewatcher.Config
}
