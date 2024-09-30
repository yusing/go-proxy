package common

import (
	"time"
)

const (
	ConnectionTimeout = 5 * time.Second
	DialTimeout       = 3 * time.Second
	KeepAlive         = 60 * time.Second
)

// file, folder structure

const (
	ConfigBasePath        = "config"
	ConfigFileName        = "config.yml"
	ConfigExampleFileName = "config.example.yml"
	ConfigPath            = ConfigBasePath + "/" + ConfigFileName

	MiddlewareDefsBasePath = ConfigBasePath + "/middlewares"
)

const (
	SchemaBasePath         = "schema"
	ConfigSchemaPath       = SchemaBasePath + "/config.schema.json"
	FileProviderSchemaPath = SchemaBasePath + "/providers.schema.json"
)

const (
	ComposeFileName        = "compose.yml"
	ComposeExampleFileName = "compose.example.yml"
)

const (
	ErrorPagesBasePath = "error_pages"
)

var (
	RequiredDirectories = []string{
		ConfigBasePath,
		SchemaBasePath,
		ErrorPagesBasePath,
	}
)

const DockerHostFromEnv = "$DOCKER_HOST"

const (
	IdleTimeoutDefault = "0"
	WakeTimeoutDefault = "30s"
	StopTimeoutDefault = "10s"
	StopMethodDefault  = "stop"
)
