package notif

import (
	"context"

	E "github.com/yusing/go-proxy/internal/error"
)

type (
	Provider interface {
		Name() string
		Send(ctx context.Context, logMsg *LogMessage) error
	}
	ProviderCreateFunc func(map[string]any) (Provider, E.Error)
	ProviderConfig     map[string]any
)

var Providers = map[string]ProviderCreateFunc{
	"gotify": newGotifyClient,
}
