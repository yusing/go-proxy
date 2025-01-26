package notif

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type Webhook struct {
	ProviderBase
	Template  string `json:"template"`
	Payload   string `json:"payload"`
	Method    string `json:"method"`
	MIMEType  string `json:"mime_type"`
	ColorMode string `json:"color_mode"`
}

//go:embed templates/discord.json
var discordPayload string

var webhookTemplates = map[string]string{
	"discord": discordPayload,
}

func (webhook *Webhook) Validate() E.Error {
	if err := webhook.ProviderBase.Validate(); err != nil && !err.Is(ErrMissingToken) {
		return err
	}

	switch webhook.MIMEType {
	case "":
		webhook.MIMEType = "application/json"
	case "application/json", "application/x-www-form-urlencoded", "text/plain":
	default:
		return E.New("invalid mime_type, expect empty, 'application/json', 'application/x-www-form-urlencoded' or 'text/plain'")
	}

	switch webhook.Template {
	case "":
		if webhook.MIMEType == "application/json" && !json.Valid([]byte(webhook.Payload)) {
			return E.New("invalid payload, expect valid JSON")
		}
		if webhook.Payload == "" {
			return E.New("invalid payload, expect non-empty")
		}
	case "discord":
		webhook.ColorMode = "dec"
		webhook.Method = http.MethodPost
		webhook.MIMEType = "application/json"
		if webhook.Payload == "" {
			webhook.Payload = discordPayload
		}
	default:
		return E.New("invalid template, expect empty or 'discord'")
	}

	switch webhook.Method {
	case "":
		webhook.Method = http.MethodPost
	case http.MethodGet, http.MethodPost, http.MethodPut:
	default:
		return E.New("invalid method, expect empty, 'GET', 'POST' or 'PUT'")
	}

	switch webhook.ColorMode {
	case "":
		webhook.ColorMode = "hex"
	case "hex", "dec":
	default:
		return E.New("invalid color_mode, expect empty, 'hex' or 'dec'")
	}

	return nil
}

// GetMethod implements Provider.
func (webhook *Webhook) GetMethod() string {
	return webhook.Method
}

// GetMIMEType implements Provider.
func (webhook *Webhook) GetMIMEType() string {
	return webhook.MIMEType
}

// makeRespError implements Provider.
func (webhook *Webhook) makeRespError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("webhook status %d, failed to read body: %w", resp.StatusCode, err)
	}
	if len(body) > 0 {
		return fmt.Errorf("webhook status %d: %s", resp.StatusCode, body)
	}
	return fmt.Errorf("webhook status %d", resp.StatusCode)
}

func (webhook *Webhook) MakeBody(logMsg *LogMessage) (io.Reader, error) {
	title, err := json.Marshal(logMsg.Title)
	if err != nil {
		return nil, err
	}
	fields, err := formatDiscord(logMsg.Extras)
	if err != nil {
		return nil, err
	}
	var color string
	if webhook.ColorMode == "hex" {
		color = logMsg.Color.HexString()
	} else {
		color = logMsg.Color.DecString()
	}
	message, err := json.Marshal(formatMarkdown(logMsg.Extras))
	if err != nil {
		return nil, err
	}
	plTempl := strings.NewReplacer(
		"$title", string(title),
		"$message", string(message),
		"$fields", fields,
		"$color", color,
	)
	var pl string
	if webhook.Template != "" {
		pl = webhookTemplates[webhook.Template]
	} else {
		pl = webhook.Payload
	}
	pl = plTempl.Replace(pl)
	return strings.NewReader(pl), nil
}
