package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type StreamRoute interface {
	Route
	ListeningUrl() string
	TargetUrl() string
	Logger() logrus.FieldLogger

	closeListeners()
	closeChannel()
	unmarkPort()
	wait()
}

type StreamRouteBase struct {
	Alias           string // to show in panel
	Type            string
	ListeningScheme string
	ListeningPort   int
	TargetScheme    string
	TargetHost      string
	TargetPort      int

	id        string
	wg        sync.WaitGroup
	stopChann chan struct{}
	l         logrus.FieldLogger
}

func newStreamRouteBase(config *ProxyConfig) (*StreamRouteBase, error) {
	var streamType string = StreamType_TCP
	var srcPort string
	var dstPort string
	var srcScheme string
	var dstScheme string

	port_split := strings.Split(config.Port, ":")
	if len(port_split) != 2 {
		cfgl.Warnf("Invalid port %s, assuming it is target port", config.Port)
		srcPort = "0"
		dstPort = config.Port
	} else {
		srcPort = port_split[0]
		dstPort = port_split[1]
	}

	if port, hasName := NamePortMap[dstPort]; hasName {
		dstPort = port
	}

	srcPortInt, err := strconv.Atoi(srcPort)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid stream source port %s, ignoring", srcPort,
		)
	}

	utils.markPortInUse(srcPortInt)

	dstPortInt, err := strconv.Atoi(dstPort)
	if err != nil {
		return nil, fmt.Errorf(
			"invalid stream target port %s, ignoring", dstPort,
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
		ListeningPort:   srcPortInt,
		TargetScheme:    dstScheme,
		TargetHost:      config.Host,
		TargetPort:      dstPortInt,

		id:        config.GetID(),
		wg:        sync.WaitGroup{},
		stopChann: make(chan struct{}),
		l: srlog.WithFields(logrus.Fields{
			"alias": config.Alias,
			"src":   fmt.Sprintf("%s://:%d", srcScheme, srcPortInt),
			"dst":   fmt.Sprintf("%s://%s:%d", dstScheme, config.Host, dstPortInt),
		}),
	}, nil
}

func NewStreamRoute(config *ProxyConfig) (StreamRoute, error) {
	switch config.Scheme {
	case StreamType_TCP:
		return NewTCPRoute(config)
	case StreamType_UDP:
		return NewUDPRoute(config)
	default:
		return nil, errors.New("unknown stream type")
	}
}

func (route *StreamRouteBase) ListeningUrl() string {
	return fmt.Sprintf("%s:%v", route.ListeningScheme, route.ListeningPort)
}

func (route *StreamRouteBase) TargetUrl() string {
	return fmt.Sprintf("%s://%s:%v", route.TargetScheme, route.TargetHost, route.TargetPort)
}

func (route *StreamRouteBase) Logger() logrus.FieldLogger {
	return route.l
}

func (route *StreamRouteBase) SetupListen() {
	if route.ListeningPort == 0 {
		freePort, err := utils.findUseFreePort(20000)
		if err != nil {
			route.l.Error(err)
			return
		}
		route.ListeningPort = freePort
		route.l.Info("Assigned free port", route.ListeningPort)
	}
	route.l.Info("Listening on", route.ListeningUrl())
}

func (route *StreamRouteBase) RemoveFromRoutes() {
	streamRoutes.Delete(route.id)
}

func (route *StreamRouteBase) wait() {
	route.wg.Wait()
}

func (route *StreamRouteBase) closeChannel() {
	close(route.stopChann)
}

func (route *StreamRouteBase) unmarkPort() {
	utils.unmarkPortInUse(route.ListeningPort)
}

func stopListening(route StreamRoute) {
	l := route.Logger()
	l.Debug("Stopping listening")
	route.closeChannel()
	route.closeListeners()

	done := make(chan struct{})

	go func() {
		route.wait()
		close(done)
		route.unmarkPort()
	}()

	select {
	case <-done:
		l.Info("Stopped listening")
		return
	case <-time.After(StreamStopListenTimeout):
		l.Error("timed out waiting for connections")
		return
	}
}
