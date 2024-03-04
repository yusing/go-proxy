package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StreamRoute interface {
	SetupListen()
	Listen()
	StopListening()
	Logf(string, ...interface{})
	PrintError(error)
	ListeningUrl() string
	TargetUrl() string

	closeListeners()
	closeChannel()
	wait()
}

type StreamRouteBase struct {
	Alias           string // to show in panel
	Type            string
	ListeningScheme string
	ListeningPort   string
	TargetScheme    string
	TargetHost      string
	TargetPort      string

	wg        sync.WaitGroup
	stopChann chan struct{}
}

func newStreamRouteBase(config *ProxyConfig) (*StreamRouteBase, error) {
	var streamType string = TCPStreamType
	var srcPort string
	var dstPort string
	var srcScheme string
	var dstScheme string

	port_split := strings.Split(config.Port, ":")
	if len(port_split) != 2 {
		log.Printf(`[Build] %s: Invalid stream port %s, `+
			`assuming it's targetPort`, config.Alias, config.Port)
		srcPort = "0"
		dstPort = config.Port
	} else {
		srcPort = port_split[0]
		dstPort = port_split[1]
	}

	port, hasName := namePortMap[dstPort]
	if hasName {
		dstPort = port
	}

	srcPortInt, err := strconv.Atoi(srcPort)
	if err != nil {
		return nil, fmt.Errorf(
			"[Build] %s: Unrecognized stream source port %s, ignoring",
			config.Alias, srcPort,
		)
	}

	utils.markPortInUse(srcPortInt)

	_, err = strconv.Atoi(dstPort)
	if err != nil {
		return nil, fmt.Errorf(
			"[Build] %s: Unrecognized stream target port %s, ignoring",
			config.Alias, dstPort,
		)
	}

	scheme_split := strings.Split(config.Scheme, ":")

	if len(scheme_split) == 2 {
		srcScheme = scheme_split[0]
		dstScheme = scheme_split[1]
	} else {
		srcScheme = config.Scheme
		dstScheme = config.Scheme
	}

	return &StreamRouteBase{
		Alias:           config.Alias,
		Type:            streamType,
		ListeningScheme: srcScheme,
		ListeningPort:   srcPort,
		TargetScheme:    dstScheme,
		TargetHost:      config.Host,
		TargetPort:      dstPort,

		wg:        sync.WaitGroup{},
		stopChann: make(chan struct{}),
	}, nil
}

func NewStreamRoute(config *ProxyConfig) (StreamRoute, error) {
	switch config.Scheme {
	case TCPStreamType:
		return NewTCPRoute(config)
	case UDPStreamType:
		return NewUDPRoute(config)
	default:
		return nil, errors.New("unknown stream type")
	}
}

func (route *StreamRouteBase) PrintError(err error) {
	if err == nil {
		return
	}
	route.Logf("Error: %s", err.Error())
}

func (route *StreamRouteBase) Logf(format string, v ...interface{}) {
	log.Printf("[%s -> %s] %s: "+format,
		append([]interface{}{
			route.ListeningScheme,
			route.TargetScheme,
			route.Alias},
			v...,
		)...,
	)
}

func (route *StreamRouteBase) ListeningUrl() string {
	return fmt.Sprintf("%s:%s", route.ListeningScheme, route.ListeningPort)
}

func (route *StreamRouteBase) TargetUrl() string {
	return fmt.Sprintf("%s://%s:%s", route.TargetScheme, route.TargetHost, route.TargetPort)
}

func (route *StreamRouteBase) SetupListen() {
	if route.ListeningPort == "0" {
		freePort, err := utils.findUseFreePort(20000)
		if err != nil {
			route.PrintError(err)
			return
		}
		route.ListeningPort = fmt.Sprintf("%d", freePort)
		route.Logf("Assigned free port %s", route.ListeningPort)
	}
	route.Logf("Listening on %s", route.ListeningUrl())
}

func (route *StreamRouteBase) wait() {
	route.wg.Wait()
}

func (route *StreamRouteBase) closeChannel() {
	close(route.stopChann)
}

func stopListening(route StreamRoute) {
	route.Logf("Stopping listening")
	route.closeChannel()
	route.closeListeners()

	done := make(chan struct{})

	go func() {
		route.wait()
		close(done)
	}()

	select {
	case <-done:
		route.Logf("Stopped listening")
		return
	case <-time.After(streamStopListenTimeout):
		route.Logf("timed out waiting for connections")
		return
	}
}

func allStreamsDo(msg string, fn ...func(StreamRoute)) {
	log.Printf("[Stream] %s", msg)

	var wg sync.WaitGroup

	for _, route := range routes.StreamRoutes.Iterator() {
		wg.Add(1)
		go func(r StreamRoute) {
			for _, f := range fn {
				f(r)
			}
			wg.Done()
		}(route)
	}

	wg.Wait()
	log.Printf("[Stream] Finished %s", msg)
}

func beginListenStreams() {
	allStreamsDo("Start", StreamRoute.SetupListen, StreamRoute.Listen)
}

func endListenStreams() {
	allStreamsDo("Stop", StreamRoute.StopListening)
}

var imageNamePortMap = map[string]string{
	"postgres":  "5432",
	"mysql":     "3306",
	"mariadb":   "3306",
	"redis":     "6379",
	"mssql":     "1433",
	"memcached": "11211",
	"rabbitmq":  "5672",
	"mongo":     "27017",
}
var extraNamePortMap = map[string]string{
	"dns":  "53",
	"ssh":  "22",
	"ftp":  "21",
	"smtp": "25",
	"pop3": "110",
	"imap": "143",
}
var namePortMap = func() map[string]string {
	m := make(map[string]string)
	for k, v := range imageNamePortMap {
		m[k] = v
	}
	for k, v := range extraNamePortMap {
		m[k] = v
	}
	return m
}()

const UDPStreamType = "udp"
const TCPStreamType = "tcp"

// const maxQueueSizePerStream = 100
const streamStopListenTimeout = 1 * time.Second
