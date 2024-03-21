package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"time"

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

const (
	certPath = "certs/cert.crt"
	keyPath  = "certs/priv.key"
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
)

const wildcardLabelPrefix = "proxy.*."

const clientUrlFromEnv = "FROM_ENV"

const (
	configPath   = "config.yml"
	templatePath = "templates/panel.html"
)

const StreamStopListenTimeout = 2 * time.Second

const udpBufferSize = 1500

var logLevel = func() logrus.Level {
	switch os.Getenv("GOPROXY_DEBUG") {
	case "1", "true":
		logrus.SetLevel(logrus.DebugLevel)
	}
	return logrus.GetLevel()
}()

var redirectHTTP = os.Getenv("GOPROXY_REDIRECT_HTTP") != "0" && os.Getenv("GOPROXY_REDIRECT_HTTP") != "false"
