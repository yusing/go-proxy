package common

import (
	"time"
)

const (
	ConnectionTimeout = 5 * time.Second
	DialTimeout       = 3 * time.Second
	KeepAlive         = 5 * time.Second
)

const (
	ProviderKind_Docker = "docker"
	ProviderKind_File   = "file"
)

// file, folder structure

const (
	ConfigBasePath = "config/"
	ConfigFileName = "config.yml"
	ConfigPath     = ConfigBasePath + ConfigFileName
)

const (
	TemplatesBasePath        = "templates/"
	PanelTemplatePath        = TemplatesBasePath + "panel/index.html"
	ConfigEditorTemplatePath = TemplatesBasePath + "config_editor/index.html"
)

const (
	SchemaBasePath      = "schema/"
	ConfigSchemaPath    = SchemaBasePath + "config.schema.json"
	ProvidersSchemaPath = SchemaBasePath + "providers.schema.json"
)

const DockerHostFromEnv = "FROM_ENV"

const (
	ProxyHTTPPort  = ":80"
	ProxyHTTPSPort = ":443"
	APIHTTPPort    = ":8888"
	PanelHTTPPort  = ":8080"
)

var WellKnownHTTPPorts = map[uint16]bool{
	80:   true,
	8000: true,
	8008: true,
	8080: true,
	3000: true,
}

var (
	ImageNamePortMapTCP = map[string]int{
		"postgres":  5432,
		"mysql":     3306,
		"mariadb":   3306,
		"redis":     6379,
		"mssql":     1433,
		"memcached": 11211,
		"rabbitmq":  5672,
		"mongo":     27017,
	}
	ExtraNamePortMapTCP = map[string]int{
		"dns":  53,
		"ssh":  22,
		"ftp":  21,
		"smtp": 25,
		"pop3": 110,
		"imap": 143,
	}
	NamePortMapTCP = func() map[string]int {
		m := make(map[string]int)
		for k, v := range ImageNamePortMapTCP {
			m[k] = v
		}
		for k, v := range ExtraNamePortMapTCP {
			m[k] = v
		}
		return m
	}()
)

// docker library uses uint16, so followed here
var ImageNamePortMapHTTP = map[string]uint16{
	"nginx":               80,
	"httpd":               80,
	"adguardhome":         3000,
	"gogs":                3000,
	"gitea":               3000,
	"portainer":           9000,
	"portainer-ce":        9000,
	"home-assistant":      8123,
	"homebridge":          8581,
	"uptime-kuma":         3001,
	"changedetection.io":  3000,
	"prometheus":          9090,
	"grafana":             3000,
	"dockge":              5001,
	"nginx-proxy-manager": 81,
}
