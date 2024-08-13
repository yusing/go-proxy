package model

import (
	"net/http"
	"strings"

	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	ProxyEntry struct {
		Alias       string      `yaml:"-" json:"-"`
		Scheme      string      `yaml:"scheme" json:"scheme"`
		Host        string      `yaml:"host" json:"host"`
		Port        string      `yaml:"port" json:"port"`
		NoTLSVerify bool        `yaml:"no_tls_verify" json:"no_tls_verify"` // http proxy only
		Path        string      `yaml:"path" json:"path"`                   // http proxy only
		SetHeaders  http.Header `yaml:"set_headers" json:"set_headers"`     // http proxy only
		HideHeaders []string    `yaml:"hide_headers" json:"hide_headers"`   // http proxy only
	}

	ProxyEntries = *F.Map[string, *ProxyEntry]
)

var NewProxyEntries = F.NewMap[string, *ProxyEntry]

func (e *ProxyEntry) SetDefaults() {
	if e.Scheme == "" {
		if strings.ContainsRune(e.Port, ':') {
			e.Scheme = "tcp"
		} else {
			switch e.Port {
			case "443", "8443":
				e.Scheme = "https"
			default:
				e.Scheme = "http"
			}
		}
	}
	if e.Path == "" {
		e.Path = "/"
	}
	switch e.Scheme {
	case "http":
		e.Port = "80"
	case "https":
		e.Port = "443"
	}
}
