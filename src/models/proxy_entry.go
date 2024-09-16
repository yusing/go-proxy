package model

import (
	"strings"

	F "github.com/yusing/go-proxy/utils/functional"
)

type (
	ProxyEntry struct {
		Alias        string            `yaml:"-" json:"-"`
		Scheme       string            `yaml:"scheme" json:"scheme"`
		Host         string            `yaml:"host" json:"host"`
		Port         string            `yaml:"port" json:"port"`
		NoTLSVerify  bool              `yaml:"no_tls_verify" json:"no_tls_verify"` // https proxy only
		PathPatterns []string          `yaml:"path_patterns" json:"path_patterns"` // http(s) proxy only
		SetHeaders   map[string]string `yaml:"set_headers" json:"set_headers"`     // http(s) proxy only
		HideHeaders  []string          `yaml:"hide_headers" json:"hide_headers"`   // http(s) proxy only
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
	if e.Host == "" {
		e.Host = "localhost"
	}
	if e.Port == "" {
		switch e.Scheme {
		case "http":
			e.Port = "80"
		case "https":
			e.Port = "443"
		}
	}
}
