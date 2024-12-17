package types

import (
	idlewatcher "github.com/yusing/go-proxy/internal/docker/idlewatcher/types"
	net "github.com/yusing/go-proxy/internal/net/types"
)

type Entry interface {
	TargetName() string
	TargetURL() net.URL
	RawEntry() *RawEntry
	IdlewatcherConfig() *idlewatcher.Config
}
