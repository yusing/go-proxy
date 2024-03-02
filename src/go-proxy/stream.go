package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

type StreamRoute struct {
	Alias           string // to show in panel
	Type            string
	ListeningScheme string
	ListeningPort   string
	TargetScheme    string
	TargetHost      string
	TargetPort      string

	Context context.Context
	Cancel  context.CancelFunc
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

func NewStreamRoute(config *ProxyConfig) (*StreamRoute, error) {
	var streamType string = TCPStreamType
	var srcPort string
	var dstPort string
	var srcScheme string
	var dstScheme string
	var srcUDPAddr *net.UDPAddr = nil
	var dstUDPAddr *net.UDPAddr = nil

	port_split := strings.Split(config.Port, ":")
	if len(port_split) != 2 {
		log.Printf(`[Build] Invalid stream port %s, `+
			`should be <listeningPort>:<targetPort>, `+
			`assuming it is targetPort`, config.Port)
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
	_, err := strconv.Atoi(dstPort)
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

	if srcScheme == "udp" {
		streamType = UDPStreamType
		srcUDPAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%s", srcPort))
		if err != nil {
			return nil, err
		}
	}

	if dstScheme == "udp" {
		streamType = UDPStreamType
		dstUDPAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", config.Host, dstPort))
		if err != nil {
			return nil, err
		}
	}

	lsPort, err := strconv.Atoi(srcPort)
	if err != nil {
		return nil, err
	}
	utils.markPortInUse(lsPort)

	ctx, cancel := context.WithCancel(context.Background())

	route := StreamRoute{
		Alias:           config.Alias,
		Type:            streamType,
		ListeningScheme: srcScheme,
		TargetScheme:    dstScheme,
		TargetHost:      config.Host,
		ListeningPort:   srcPort,
		TargetPort:      dstPort,

		Context: ctx,
		Cancel:  cancel,
	}

	if streamType == TCPStreamType {
		return &route, nil
	}
	
	return (*StreamRoute)(unsafe.Pointer(&UDPRoute{
		StreamRoute:   route,
		ConnMap:       make(map[net.Addr]*net.UDPConn),
		ConnMapMutex:  sync.Mutex{},
		QueueSize:     atomic.Int32{},
		SourceUDPAddr: srcUDPAddr,
		TargetUDPAddr: dstUDPAddr,
	})), nil
}

func (route *StreamRoute) PrintError(err error) {
	if err == nil {
		return
	}
	log.Printf("[Stream] %s (%s => %s) error: %v", route.Alias, route.ListeningUrl(), route.TargetUrl(), err)
}

func (route *StreamRoute) ListeningUrl() string {
	return fmt.Sprintf("%s:%s", route.ListeningScheme, route.ListeningPort)
}

func (route *StreamRoute) TargetUrl() string {
	return fmt.Sprintf("%s://%s:%s", route.TargetScheme, route.TargetHost, route.TargetPort)
}

func (route *StreamRoute) listenStream() {
	if route.ListeningPort == "0" {
		freePort, err := utils.findFreePort(20000)
		if err != nil {
			route.PrintError(err)
			return
		}
		route.ListeningPort = fmt.Sprintf("%d", freePort)
		utils.markPortInUse(freePort)
	}
	if route.Type == UDPStreamType {
		listenUDP((*UDPRoute)(unsafe.Pointer(route)))
	} else {
		listenTCP(route)
	}
}

func beginListenStreams() {
	for _, route := range routes.StreamRoutes {
		go route.listenStream()
	}
}

func endListenStreams() {
	var wg sync.WaitGroup
	wg.Add(len(routes.StreamRoutes))
	defer wg.Wait()

	routes.Mutex.Lock()
	defer routes.Mutex.Unlock()

	for _, route := range routes.StreamRoutes {
		go func(r *StreamRoute) {
			r.Cancel()
			wg.Done()
		}(route)
	}
}
