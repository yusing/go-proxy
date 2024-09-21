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

const DockerHostFromEnv = "$DOCKER_HOST"

const (
	ProxyHTTPPort  = ":80"
	ProxyHTTPSPort = ":443"
	APIHTTPPort    = ":8888"
)

var WellKnownHTTPPorts = map[uint16]bool{
	80:   true,
	8000: true,
	8008: true,
	8080: true,
	3000: true,
}

var (
	ServiceNamePortMapTCP = map[string]int{
		"postgres":         5432,
		"mysql":            3306,
		"mariadb":          3306,
		"redis":            6379,
		"mssql":            1433,
		"memcached":        11211,
		"rabbitmq":         5672,
		"mongo":            27017,
		"minecraft-server": 25565,

		"dns":  53,
		"ssh":  22,
		"ftp":  21,
		"smtp": 25,
		"pop3": 110,
		"imap": 143,
	}
)

var ImageNamePortMapHTTP = map[string]int{
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

const (
	IdleTimeoutDefault = "0"
	WakeTimeoutDefault = "10s"
	StopTimeoutDefault = "10s"
	StopMethodDefault  = "stop"
)
