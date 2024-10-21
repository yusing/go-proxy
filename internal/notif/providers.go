package notif

import (
	"context"

	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/internal/error"
)

type (
	Provider interface {
		Name() string
		Send(ctx context.Context, entry *logrus.Entry) error
	}
	ProviderCreateFunc func(map[string]any) (Provider, E.Error)
	ProviderConfig     map[string]any
)

var Providers = map[string]ProviderCreateFunc{
	"gotify": newGotifyClient,
}
