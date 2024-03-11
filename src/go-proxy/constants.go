package main

import "time"

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

const configPath = "config.yml"

const StreamStopListenTimeout = 1 * time.Second

const templateFile = "templates/panel.html"

const udpBufferSize = 1500
