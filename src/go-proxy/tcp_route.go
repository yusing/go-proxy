package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

const tcpDialTimeout = 5 * time.Second

type Pipes []*BidirectionalPipe

type TCPRoute struct {
	*StreamRouteBase
	listener net.Listener
	pipe     Pipes
	mu       sync.Mutex
}

func NewTCPRoute(base *StreamRouteBase) StreamImpl {
	return &TCPRoute{
		StreamRouteBase: base,
		listener:        nil,
		pipe:            make(Pipes, 0),
	}
}

func (route *TCPRoute) Setup() error {
	in, err := net.Listen("tcp", fmt.Sprintf(":%v", route.ListeningPort))
	if err != nil {
		return err
	}
	route.listener = in
	return nil
}

func (route *TCPRoute) Accept() (interface{}, error) {
	return route.listener.Accept()
}

func (route *TCPRoute) Handle(c interface{}) error {
	clientConn := c.(net.Conn)

	defer clientConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), tcpDialTimeout)
	defer cancel()

	serverAddr := fmt.Sprintf("%s:%v", route.TargetHost, route.TargetPort)
	dialer := &net.Dialer{}

	serverConn, err := dialer.DialContext(ctx, route.TargetScheme, serverAddr)
	if err != nil {
		return err
	}

	pipeCtx, pipeCancel := context.WithCancel(context.Background())
	go func() {
		<-route.stopCh
		pipeCancel()
	}()

	route.mu.Lock()
	pipe := NewBidirectionalPipe(pipeCtx, clientConn, serverConn)
	route.pipe = append(route.pipe, pipe)
	route.mu.Unlock()
	return pipe.Start()
}

func (route *TCPRoute) CloseListeners() {
	if route.listener == nil {
		return
	}
	route.listener.Close()
	route.listener = nil
	for _, pipe := range route.pipe {
		if err := pipe.Stop(); err != nil {
			route.l.Error(err)
		}
	}
}
