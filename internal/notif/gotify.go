package notif

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gotify/server/v2/model"
	"github.com/rs/zerolog"
	E "github.com/yusing/go-proxy/internal/error"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
	GotifyClient struct {
		GotifyConfig

		url  *url.URL
		http http.Client
	}
	GotifyConfig struct {
		URL   string `json:"url" yaml:"url"`
		Token string `json:"token" yaml:"token"`
	}
	GotifyMessage model.Message
)

const gotifyMsgEndpoint = "/message"

func newGotifyClient(cfg map[string]any) (Provider, E.Error) {
	client := new(GotifyClient)
	err := U.Deserialize(cfg, &client.GotifyConfig)
	if err != nil {
		return nil, err
	}

	url, uErr := url.Parse(client.URL)
	if uErr != nil {
		return nil, E.Errorf("invalid gotify URL %s", client.URL)
	}

	client.url = url
	return client, err
}

// Name implements NotifProvider.
func (client *GotifyClient) Name() string {
	return "gotify"
}

// Send implements NotifProvider.
func (client *GotifyClient) Send(ctx context.Context, logMsg *LogMessage) error {
	var priority int

	switch logMsg.Level {
	case zerolog.WarnLevel:
		priority = 2
	case zerolog.ErrorLevel:
		priority = 5
	case zerolog.FatalLevel, zerolog.PanicLevel:
		priority = 8
	}

	msg := &GotifyMessage{
		Title:    logMsg.Title,
		Message:  logMsg.Message,
		Priority: priority,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.url.String()+gotifyMsgEndpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.Token)

	resp, err := client.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send gotify message: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errm model.Error
		err = json.NewDecoder(resp.Body).Decode(&errm)
		if err != nil {
			return fmt.Errorf("gotify status %d, but failed to decode err response: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("gotify status %d %s: %s", resp.StatusCode, errm.Error, errm.ErrorDescription)
	}
	return nil
}
