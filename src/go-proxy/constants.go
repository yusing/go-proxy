package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/santhosh-tekuri/jsonschema"
	"github.com/sirupsen/logrus"
)

var (
	ImageNamePortMap = map[string]string{
		"postgres":  "5432",
		"mysql":     "3306",
		"mariadb":   "3306",
		"redis":     "6379",
		"mssql":     "1433",
		"memcached": "11211",
		"rabbitmq":  "5672",
		"mongo":     "27017",
	}
	ExtraNamePortMap = map[string]string{
		"dns":  "53",
		"ssh":  "22",
		"ftp":  "21",
		"smtp": "25",
		"pop3": "110",
		"imap": "143",
	}
	NamePortMap = func() map[string]string {
		m := make(map[string]string)
		for k, v := range ImageNamePortMap {
			m[k] = v
		}
		for k, v := range ExtraNamePortMap {
			m[k] = v
		}
		return m
	}()
)

var wellKnownHTTPPorts = map[uint16]bool{
	80:   true,
	8000: true,
	8008: true, // alternative HTTP port
	8080: true,
	3000: true, // adguardhome, gogs, etc
}

var (
	StreamSchemes = []string{StreamType_TCP, StreamType_UDP} // TODO: support "tcp:udp", "udp:tcp"
	HTTPSchemes   = []string{"http", "https"}
	ValidSchemes  = append(StreamSchemes, HTTPSchemes...)
)

const (
	StreamType_UDP = "udp"
	StreamType_TCP = "tcp"
)

const (
	ProxyPathMode_Forward     = "forward"
	ProxyPathMode_Sub         = "sub"
	ProxyPathMode_RemovedPath = ""
)

const (
	ProviderKind_Docker = "docker"
	ProviderKind_File   = "file"
)

// TODO: default + per proxy
var (
	transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
	}

	transportNoTLS = func() *http.Transport {
		var clone = transport.Clone()
		clone.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return clone
	}()

	healthCheckHttpClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
			ForceAttemptHTTP2: true,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
)

const wildcardLabelPrefix = "proxy.*."

const clientUrlFromEnv = "FROM_ENV"

const (
	certBasePath    = "certs/"
	certFileDefault = certBasePath + "cert.crt"
	keyFileDefault  = certBasePath + "priv.key"

	configBasePath = "config/"
	configPath     = configBasePath + "config.yml"

	templatesBasePath        = "templates/"
	panelTemplatePath        = templatesBasePath + "panel/index.html"
	configEditorTemplatePath = templatesBasePath + "config_editor/index.html"

	schemaBasePath      = "schema/"
	configSchemaPath    = schemaBasePath + "config.schema.json"
	providersSchemaPath = schemaBasePath + "providers.schema.json"
)

var (
	configSchema    *jsonschema.Schema
	providersSchema *jsonschema.Schema
	_               = func() *jsonschema.Compiler {
		c := jsonschema.NewCompiler()
		c.Draft = jsonschema.Draft7
		var err error
		if configSchema, err = c.Compile(configSchemaPath); err != nil {
			panic(err)
		}
		if providersSchema, err = c.Compile(providersSchemaPath); err != nil {
			panic(err)
		}
		return c
	}()
)

const (
	streamStopListenTimeout = 1 * time.Second
	streamDialTimeout       = 3 * time.Second
)

const udpBufferSize = 1500

var logLevel = func() logrus.Level {
	switch os.Getenv("GOPROXY_DEBUG") {
	case "1", "true":
		logrus.SetLevel(logrus.DebugLevel)
	}
	return logrus.GetLevel()
}()
