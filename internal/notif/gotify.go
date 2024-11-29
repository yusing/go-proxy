package notif

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gotify/server/v2/model"
	"github.com/rs/zerolog"
)

type (
	GotifyClient struct {
		N   string `json:"name" validate:"required"`
		U   string `json:"url" validate:"url"`
		Tok string `json:"token" validate:"required"`
	}
	GotifyMessage model.MessageExternal
)

const gotifyMsgEndpoint = "/message"

// Name implements Provider.
func (client *GotifyClient) Name() string {
	return client.N
}

// Method implements Provider.
func (client *GotifyClient) Method() string {
	return http.MethodPost
}

// URL implements Provider.
func (client *GotifyClient) URL() string {
	return client.U + gotifyMsgEndpoint
}

// Token implements Provider.
func (client *GotifyClient) Token() string {
	return client.Tok
}

// MIMEType implements Provider.
func (client *GotifyClient) MIMEType() string {
	return "application/json"
}

// MakeBody implements Provider.
func (client *GotifyClient) MakeBody(logMsg *LogMessage) (io.Reader, error) {
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
		Message:  formatMarkdown(logMsg.Extras),
		Priority: &priority,
		Extras: map[string]interface{}{
			"client::display": map[string]string{
				"contentType": "text/markdown",
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

// makeRespError implements Provider.
func (client *GotifyClient) makeRespError(resp *http.Response) error {
	var errm model.Error
	err := json.NewDecoder(resp.Body).Decode(&errm)
	if err != nil {
		return fmt.Errorf(ProviderGotify+" status %d, but failed to decode err response: %w", resp.StatusCode, err)
	}
	return fmt.Errorf(ProviderGotify+" status %d %s: %s", resp.StatusCode, errm.Error, errm.ErrorDescription)
}
