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

	l := srlog.WithFields(logrus.Fields{
		"alias": config.Alias,
	})
	portSplit := strings.Split(config.Port, ":")
	if len(portSplit) != 2 {
		l.Warnf(
			`%s: invalid port %s, 
			assuming it is target port`,
			config.Alias,
			config.Port,
		)
		srcPort = "0" // will assign later
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

	if srcScheme != dstScheme {
		return nil, NewNestedError("unsupported").Subjectf("%v -> %v", srcScheme, dstScheme)
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
		l:       l,
	}, nil
}

func NewStreamRoute(config *ProxyConfig) (StreamRoute, error) {
	base, err := newStreamRouteBase(config)
	if err != nil {
		return nil, err
	}
	switch config.Scheme {
	case StreamType_TCP:
		base.StreamImpl = NewTCPRoute(base)
	case StreamType_UDP:
		base.StreamImpl = NewUDPRoute(base)
	default:
		return nil, NewNestedError("invalid stream type").Subject(config.Scheme)
	}
	return base, nil
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
	route.wg.Wait()
	route.ensurePort()
	if err := route.Setup(); err != nil {
		route.l.Errorf("failed to setup: %v", err)
		return
	}
	route.started = true
	streamRoutes.Set(route.id, route)
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

// id    -> target
type StreamRoutes SafeMap[string, StreamRoute]

var streamRoutes StreamRoutes = NewSafeMapOf[StreamRoutes]()
