package notif

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/yusing/go-proxy/internal/utils"
)

type Webhook struct {
	N        string `json:"name" validate:"required"`
	U        string `json:"url" validate:"url"`
	Template string `json:"template" validate:"omitempty,oneof=discord"`
	Payload  string `json:"payload" validate:"jsonIfTemplateNotUsed"`
	Tok      string `json:"token"`
	Meth     string `json:"method" validate:"omitempty,oneof=GET POST PUT"`
	MIMETyp  string `json:"mime_type"`
	ColorM   string `json:"color_mode" validate:"omitempty,oneof=hex dec"`
}

//go:embed templates/discord.json
var discordPayload string

var webhookTemplates = map[string]string{
	"discord": discordPayload,
}

func jsonIfTemplateNotUsed(fl validator.FieldLevel) bool {
	template := fl.Parent().FieldByName("Template").String()
	if template != "" {
		return true
	}
	payload := fl.Field().String()
	return json.Valid([]byte(payload))
}

func init() {
	utils.Validator().RegisterValidation("jsonIfTemplateNotUsed", jsonIfTemplateNotUsed)
}

// Name implements Provider.
func (webhook *Webhook) Name() string {
	return webhook.N
}

// Method implements Provider.
func (webhook *Webhook) Method() string {
	if webhook.Meth != "" {
		return webhook.Meth
	} else {
		return http.MethodPost
	}
}

// URL implements Provider.
func (webhook *Webhook) URL() string {
	return webhook.U
}

// Token implements Provider.
func (webhook *Webhook) Token() string {
	return webhook.Tok
}

// MIMEType implements Provider.
func (webhook *Webhook) MIMEType() string {
	if webhook.MIMETyp != "" {
		return webhook.MIMETyp
	} else {
		return "application/json"
	}
}

func (Webhook *Webhook) ColorMode() string {
	switch Webhook.Template {
	case "discord":
		return "dec"
	default:
		return Webhook.ColorM
	}
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
	if webhook.ColorMode() == "hex" {
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
		"$fields", string(fields),
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
