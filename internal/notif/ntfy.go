package notif

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	E "github.com/yusing/go-proxy/internal/error"
)

// See https://docs.ntfy.sh/publish
type Ntfy struct {
	ProviderBase
	Topic string    `json:"topic"`
	Style NtfyStyle `json:"style"`
}

type NtfyStyle string

const (
	NtfyStyleMarkdown NtfyStyle = "markdown"
	NtfyStylePlain    NtfyStyle = "plain"
)

func (n *Ntfy) Validate() E.Error {
	if n.URL == "" {
		return E.New("url is required")
	}
	if n.Topic == "" {
		return E.New("topic is required")
	}
	if n.Topic[0] == '/' {
		return E.New("topic should not start with a slash")
	}
	switch n.Style {
	case "":
		n.Style = NtfyStyleMarkdown
	case NtfyStyleMarkdown, NtfyStylePlain:
	default:
		return E.Errorf("invalid style, expecting %q or %q, got %q", NtfyStyleMarkdown, NtfyStylePlain, n.Style)
	}
	return nil
}

func (n *Ntfy) GetURL() string {
	if n.URL[len(n.URL)-1] == '/' {
		return n.URL + n.Topic
	}
	return n.URL + "/" + n.Topic
}

func (n *Ntfy) GetMIMEType() string {
	return ""
}

func (n *Ntfy) GetToken() string {
	return n.Token
}

func (n *Ntfy) MakeBody(logMsg *LogMessage) (io.Reader, error) {
	switch n.Style {
	case NtfyStyleMarkdown:
		return strings.NewReader(formatMarkdown(logMsg.Extras)), nil
	default:
		return &bytes.Buffer{}, nil
	}
}

func (n *Ntfy) SetHeaders(logMsg *LogMessage, headers http.Header) {
	headers.Set("Title", logMsg.Title)

	switch logMsg.Level {
	// warning (or other unspecified) uses default priority
	case zerolog.FatalLevel:
		headers.Set("Priority", "urgent")
	case zerolog.ErrorLevel:
		headers.Set("Priority", "high")
	case zerolog.InfoLevel:
		headers.Set("Priority", "low")
	case zerolog.DebugLevel:
		headers.Set("Priority", "min")
	}

	if n.Style == NtfyStyleMarkdown {
		headers.Set("Markdown", "yes")
	}
}
