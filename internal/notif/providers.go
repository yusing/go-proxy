package notif

import (
	"context"
	"io"
	"net/http"

	E "github.com/yusing/go-proxy/internal/error"
	gphttp "github.com/yusing/go-proxy/internal/net/http"
	"github.com/yusing/go-proxy/internal/utils"
)

type (
	Provider interface {
		utils.CustomValidator

		GetName() string
		GetURL() string
		GetToken() string
		GetMethod() string
		GetMIMEType() string

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

func notifyProvider(ctx context.Context, provider Provider, msg *LogMessage) error {
	body, err := provider.MakeBody(msg)
	if err != nil {
		return E.PrependSubject(provider.GetName(), err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		provider.GetURL(),
		body,
	)
	if err != nil {
		return E.PrependSubject(provider.GetName(), err)
	}

	req.Header.Set("Content-Type", provider.GetMIMEType())
	if provider.GetToken() != "" {
		req.Header.Set("Authorization", "Bearer "+provider.GetToken())
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return E.PrependSubject(provider.GetName(), err)
	}

	defer resp.Body.Close()

	if !gphttp.IsSuccess(resp.StatusCode) {
		return provider.makeRespError(resp)
	}
	return nil
}
