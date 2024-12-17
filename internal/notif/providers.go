package notif

import (
	"context"
	"fmt"
	"io"
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
	Provider interface {
		Name() string
		URL() string
		Method() string
		Token() string
		MIMEType() string
		MakeBody(logMsg *LogMessage) (io.Reader, error)

		makeRespError(resp *http.Response) error
	}
	ProviderCreateFunc func(map[string]any) (Provider, E.Error)
	ProviderConfig     map[string]any
)

const (
	ProviderGotify  = "gotify"
	ProviderWebhook = "webhook"
)

var Providers = map[string]ProviderCreateFunc{
	ProviderGotify:  newNotifProvider[*GotifyClient],
	ProviderWebhook: newNotifProvider[*Webhook],
}

func newNotifProvider[T Provider](cfg map[string]any) (Provider, E.Error) {
	var client T
	err := U.Deserialize(cfg, &client)
	if err != nil {
		return nil, err.Subject(client.Name())
	}
	return client, nil
}

func formatError(p Provider, err error) error {
	return fmt.Errorf("%s error: %w", p.Name(), err)
}

func notifyProvider(ctx context.Context, provider Provider, msg *LogMessage) error {
	body, err := provider.MakeBody(msg)
	if err != nil {
		return formatError(provider, err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		provider.URL(),
		body,
	)
	if err != nil {
		return formatError(provider, err)
	}

	req.Header.Set("Content-Type", provider.MIMEType())
	if provider.Token() != "" {
		req.Header.Set("Authorization", "Bearer "+provider.Token())
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return formatError(provider, err)
	}

	defer resp.Body.Close()

	if !gphttp.IsSuccess(resp.StatusCode) {
		return provider.makeRespError(resp)
	}
	return nil
}
