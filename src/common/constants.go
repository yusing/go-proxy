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
	SchemaBasePath         = "schema/"
	ConfigSchemaPath       = SchemaBasePath + "config.schema.json"
	FileProviderSchemaPath = SchemaBasePath + "providers.schema.json"
)

const DockerHostFromEnv = "$DOCKER_HOST"

const (
	IdleTimeoutDefault = "0"
	WakeTimeoutDefault = "10s"
	StopTimeoutDefault = "10s"
	StopMethodDefault  = "stop"
)
