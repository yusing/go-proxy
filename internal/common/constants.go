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
	DotEnvPath        = ".env"
	DotEnvExamplePath = ".env.example"

	ConfigBasePath        = "config"
	ConfigFileName        = "config.yml"
	ConfigExampleFileName = "config.example.yml"
	ConfigPath            = ConfigBasePath + "/" + ConfigFileName

	JWTKeyPath = ConfigBasePath + "/jwt.key"

	MiddlewareComposeBasePath = ConfigBasePath + "/middlewares"

	SchemaBasePath         = "schema"
	ConfigSchemaPath       = SchemaBasePath + "/config.schema.json"
	FileProviderSchemaPath = SchemaBasePath + "/providers.schema.json"

	ComposeFileName        = "compose.yml"
	ComposeExampleFileName = "compose.example.yml"

	ErrorPagesBasePath = "error_pages"
)

var RequiredDirectories = []string{
	ConfigBasePath,
	SchemaBasePath,
	ErrorPagesBasePath,
	MiddlewareComposeBasePath,
}

const DockerHostFromEnv = "$DOCKER_HOST"

const (
	HealthCheckIntervalDefault = 5 * time.Second
	HealthCheckTimeoutDefault  = 5 * time.Second

	WakeTimeoutDefault = "30s"
	StopTimeoutDefault = "10s"
	StopMethodDefault  = "stop"
)

const HeaderCheckRedirect = "X-Goproxy-Check-Redirect"
