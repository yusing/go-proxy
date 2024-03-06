package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
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
		glog.Infof(`[Build] %s: Invalid stream port %s, `+
			`assuming it's targetPort`, config.Alias, config.Port)
		srcPort = "0"
		dstPort = config.Port
	} else {
		srcPort = port_split[0]
		dstPort = port_split[1]
	}

	port, hasName := NamePortMap[dstPort]
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
	glog.Errorf("[%s -> %s] %s: %v",
		route.ListeningScheme,
		route.TargetScheme,
		route.Alias,
		err,
	)
}

func (route *StreamRouteBase) Logf(format string, v ...interface{}) {
	glog.Infof("[%s -> %s] %s: "+format,
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
	case <-time.After(StreamStopListenTimeout):
		route.Logf("timed out waiting for connections")
		return
	}
}

func allStreamsDo(msg string, fn ...func(StreamRoute)) {
	glog.Infof("[Stream] %s", msg)

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
	glog.Infof("[Stream] Finished %s", msg)
}

func BeginListenStreams() {
	allStreamsDo("Start", StreamRoute.SetupListen, StreamRoute.Listen)
}

func EndListenStreams() {
	allStreamsDo("Stop", StreamRoute.StopListening)
}
