package notif

import (
	"testing"

	"github.com/yusing/go-proxy/internal/utils"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestGotifyValidation(t *testing.T) {
	t.Parallel()

	newGotify := Providers[ProviderGotify]

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		_, err := newGotify(map[string]any{
			"name":  "test",
			"url":   "https://example.com",
			"token": "token",
		})
		ExpectNoError(t, err)
	})

	t.Run("missing url", func(t *testing.T) {
		t.Parallel()
		_, err := newGotify(map[string]any{
			"name":  "test",
			"token": "token",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})

	t.Run("missing token", func(t *testing.T) {
		t.Parallel()
		_, err := newGotify(map[string]any{
			"name": "test",
			"url":  "https://example.com",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})

	t.Run("invalid url", func(t *testing.T) {
		t.Parallel()
		_, err := newGotify(map[string]any{
			"name":  "test",
			"url":   "example.com",
			"token": "token",
		})
		ExpectError(t, utils.ErrValidationError, err)
	})
}
