package main

import (
	"context"
	"fmt"
	"net"
	"time"
)

const tcpDialTimeout = 5 * time.Second

type Pipes []*BidirectionalPipe

type TCPRoute struct {
	*StreamRouteBase
	listener net.Listener
}

func NewTCPRoute(config *ProxyConfig) (StreamRoute, error) {
	base, err := newStreamRouteBase(config)
	if err != nil {
		return nil, NewNestedErrorFrom(err).Subject(config.Alias)
	}
	if base.TargetScheme != StreamType_TCP {
		return nil, NewNestedError("unsupported").Subjectf("tcp -> %s", base.TargetScheme)
	}
	base.StreamImpl = &TCPRoute{
		StreamRouteBase: base,
		listener:        nil,
	}
	return base, nil
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

func (route *TCPRoute) HandleConnection(c interface{}) error {
	clientConn := c.(net.Conn)

	defer clientConn.Close()
	defer route.wg.Done()

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
	pipe := NewBidirectionalPipe(pipeCtx, clientConn, serverConn)
	pipe.Start()
	pipe.Wait()
	pipe.Close()
	return nil
}

func (route *TCPRoute) CloseListeners() {
	if route.listener == nil {
		return
	}
	route.listener.Close()
	route.listener = nil
}
