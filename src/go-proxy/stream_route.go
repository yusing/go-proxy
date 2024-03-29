package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type StreamImpl interface {
	Setup() error
	Accept() (interface{}, error)
	Handle(interface{}) error
	CloseListeners()
}

type StreamRoute interface {
	Route
	ListeningUrl() string
	TargetUrl() string
	Logger() logrus.FieldLogger
}

type StreamRouteBase struct {
	Alias           string // to show in panel
	Type            string
	ListeningScheme string
	ListeningPort   int
	TargetScheme    string
	TargetHost      string
	TargetPort      int

	id      string
	wg      sync.WaitGroup
	stopCh  chan struct{}
	connCh  chan interface{}
	started bool
	l       logrus.FieldLogger

	StreamImpl
}

func newStreamRouteBase(config *ProxyConfig) (*StreamRouteBase, error) {
	var streamType string = StreamType_TCP
	var srcPort, dstPort string
	var srcScheme, dstScheme string

	portSplit := strings.Split(config.Port, ":")
	if len(portSplit) != 2 {
		cfgl.Warnf("invalid port %s, assuming it is target port", config.Port)
		srcPort = "0"
		dstPort = config.Port
	} else {
		srcPort = portSplit[0]
		dstPort = portSplit[1]
	}

	if port, hasName := NamePortMapTCP[dstPort]; hasName {
		dstPort = port
	}

	srcPortInt, err := strconv.Atoi(srcPort)
	if err != nil {
		return nil, NewNestedError("invalid stream source port").Subject(srcPort)
	}

	utils.markPortInUse(srcPortInt)

	dstPortInt, err := strconv.Atoi(dstPort)
	if err != nil {
		return nil, NewNestedError("invalid stream target port").Subject(dstPort)
	}

	schemeSplit := strings.Split(config.Scheme, ":")
	if len(schemeSplit) == 2 {
		srcScheme = schemeSplit[0]
		dstScheme = schemeSplit[1]
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

		id:      config.GetID(),
		wg:      sync.WaitGroup{},
		stopCh:  make(chan struct{}, 1),
		connCh:  make(chan interface{}),
		started: false,
		l: srlog.WithFields(logrus.Fields{
			"alias": config.Alias,
			// "src":   fmt.Sprintf("%s://:%d", srcScheme, srcPortInt),
			// "dst":   fmt.Sprintf("%s://%s:%d", dstScheme, config.Host, dstPortInt),
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
		return nil, NewNestedError("invalid stream type").Subject(config.Scheme)
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

func (route *StreamRouteBase) Start() {
	route.ensurePort()
	if err := route.Setup(); err != nil {
		route.l.Errorf("failed to setup: %v", err)
		return
	}
	route.started = true
	route.wg.Add(2)
	go route.grAcceptConnections()
	go route.grHandleConnections()
}

func (route *StreamRouteBase) Stop() {
	if !route.started {
		return
	}
	l := route.Logger()
	l.Debug("stopping listening")
	close(route.stopCh)
	route.CloseListeners()

	done := make(chan struct{}, 1)
	go func() {
		route.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		l.Info("stopped listening")
	case <-time.After(streamStopListenTimeout):
		l.Error("timed out waiting for connections")
	}

	utils.unmarkPortInUse(route.ListeningPort)
	streamRoutes.Delete(route.id)
}

func (route *StreamRouteBase) ensurePort() {
	if route.ListeningPort == 0 {
		freePort, err := utils.findUseFreePort(20000)
		if err != nil {
			route.l.Error(err)
			return
		}
		route.ListeningPort = freePort
		route.l.Info("listening on free port ", route.ListeningPort)
		return
	}
	route.l.Info("listening on ", route.ListeningUrl())
}

func (route *StreamRouteBase) grAcceptConnections() {
	defer route.wg.Done()

	for {
		select {
		case <-route.stopCh:
			return
		default:
			conn, err := route.Accept()
			if err != nil {
				select {
				case <-route.stopCh:
					return
				default:
					route.l.Error(err)
					continue
				}
			}
			route.connCh <- conn
		}
	}
}

func (route *StreamRouteBase) grHandleConnections() {
	defer route.wg.Done()

	for {
		select {
		case <-route.stopCh:
			return
		case conn := <-route.connCh:
			go func() {
				err := route.Handle(conn)
				if err != nil {
					route.l.Error(err)
				}
			}()
		}
	}
}
