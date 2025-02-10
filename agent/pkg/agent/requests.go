package agent

import (
	"io"
	"net/http"

	"github.com/coder/websocket"
	"github.com/yusing/go-proxy/internal/logging"
	"golang.org/x/net/context"
)

func (cfg *AgentConfig) Do(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, APIBaseURL+endpoint, body)
	logging.Debug().Msgf("request: %s %s", method, req.URL.String())
	if err != nil {
		return nil, err
	}
	return cfg.httpClient.Do(req)
}

func (cfg *AgentConfig) Fetch(ctx context.Context, endpoint string) ([]byte, int, error) {
	resp, err := cfg.Do(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

func (cfg *AgentConfig) Websocket(ctx context.Context, endpoint string) (*websocket.Conn, *http.Response, error) {
	return websocket.Dial(ctx, APIBaseURL+endpoint, &websocket.DialOptions{
		HTTPClient: cfg.NewHTTPClient(),
		Host:       AgentHost,
	})
}
