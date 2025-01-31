package notif

import (
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils"
)

type NotificationConfig struct {
	ProviderName string   `json:"provider"`
	Provider     Provider `json:"-"`
}

var (
	ErrMissingNotifProvider     = E.New("missing notification provider")
	ErrInvalidNotifProviderType = E.New("invalid notification provider type")
	ErrUnknownNotifProvider     = E.New("unknown notification provider")
)

// UnmarshalMap implements MapUnmarshaler.
func (cfg *NotificationConfig) UnmarshalMap(m map[string]any) (err E.Error) {
	// extract provider name
	providerName := m["provider"]
	switch providerName := providerName.(type) {
	case string:
		cfg.ProviderName = providerName
	default:
		return ErrInvalidNotifProviderType
	}
	delete(m, "provider")

	if cfg.ProviderName == "" {
		return ErrMissingNotifProvider
	}

	// validate provider name and initialize provider
	switch cfg.ProviderName {
	case ProviderWebhook:
		cfg.Provider = &Webhook{}
	case ProviderGotify:
		cfg.Provider = &GotifyClient{}
	case ProviderNtfy:
		cfg.Provider = &Ntfy{}
	default:
		return ErrUnknownNotifProvider.
			Subject(cfg.ProviderName).
			Withf("expect %s or %s", ProviderWebhook, ProviderGotify)
	}

	// unmarshal provider config
	if err := utils.Deserialize(m, cfg.Provider); err != nil {
		return err
	}

	// validate provider
	return cfg.Provider.Validate()
}
