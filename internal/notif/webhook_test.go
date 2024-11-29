package notif

import (
	"encoding/json"
	"testing"

	"github.com/yusing/go-proxy/internal/utils"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestWebhookValidation(t *testing.T) {
	t.Parallel()

	newWebhook := Providers[ProviderWebhook]

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		_, err := newWebhook(map[string]any{
			"name":    "test",
			"url":     "https://example.com",
			"payload": "{}",
		})
		ExpectNoError(t, err)
	})
	t.Run("valid template", func(t *testing.T) {
		t.Parallel()
		_, err := newWebhook(map[string]any{
			"name":     "test",
			"url":      "https://example.com",
			"template": "discord",
		})
		ExpectNoError(t, err)
	})

	t.Run("missing url", func(t *testing.T) {
		t.Parallel()
		_, err := newWebhook(map[string]any{
			"name":    "test",
			"payload": "{}",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})

	t.Run("missing payload", func(t *testing.T) {
		t.Parallel()
		_, err := newWebhook(map[string]any{
			"name": "test",
			"url":  "https://example.com",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})
	t.Run("invalid url", func(t *testing.T) {
		t.Parallel()
		_, err := newWebhook(map[string]any{
			"name":    "test",
			"url":     "example.com",
			"payload": "{}",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})
	t.Run("invalid payload", func(t *testing.T) {
		t.Parallel()
		_, err := newWebhook(map[string]any{
			"name":    "test",
			"url":     "https://example.com",
			"payload": "abcd",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})
	t.Run("invalid method", func(t *testing.T) {
		t.Parallel()
		_, err := newWebhook(map[string]any{
			"name":    "test",
			"url":     "https://example.com",
			"payload": "{}",
			"method":  "abcd",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})
	t.Run("invalid template", func(t *testing.T) {
		t.Parallel()
		_, err := newWebhook(map[string]any{
			"name":     "test",
			"url":      "https://example.com",
			"template": "abcd",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})
}

func TestWebhookBody(t *testing.T) {
	t.Parallel()

	var webhook Webhook
	webhook.Payload = discordPayload
	bodyReader, err := webhook.MakeBody(&LogMessage{
		Title: "abc",
		Extras: map[string]any{
			"foo": "bar",
		},
	})
	ExpectNoError(t, err)

	var body map[string][]map[string]any
	err = json.NewDecoder(bodyReader).Decode(&body)
	ExpectNoError(t, err)

	ExpectEqual(t, body["embeds"][0]["title"], "abc")
	fields := ExpectType[[]map[string]any](t, body["embeds"][0]["fields"])
	ExpectEqual(t, fields[0]["name"], "foo")
	ExpectEqual(t, fields[0]["value"], "bar")
}
