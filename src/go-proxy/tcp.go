package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const tcpDialTimeout = 5 * time.Second

type TCPRoute struct {
	*StreamRouteBase
	listener net.Listener
	connChan chan net.Conn
}

func NewTCPRoute(config *ProxyConfig) (StreamRoute, error) {
	base, err := newStreamRouteBase(config)
	if err != nil {
		return nil, err
	}
	if base.TargetScheme != TCPStreamType {
		return nil, fmt.Errorf("tcp to %s not yet supported", base.TargetScheme)
	}
	return &TCPRoute{
		StreamRouteBase: base,
		listener:        nil,
		connChan:        make(chan net.Conn),
	}, nil
}

func (route *TCPRoute) Listen() {
	in, err := net.Listen("tcp", ":"+route.ListeningPort)
	if err != nil {
		route.PrintError(err)
		return
	}
	route.listener = in
	route.wg.Add(2)
	go route.grAcceptConnections()
	go route.grHandleConnections()
}

func (route *TCPRoute) StopListening() {
	stopListening(route)
}

func (route *TCPRoute) closeListeners() {
	if route.listener == nil {
		return
	}
	route.listener.Close()
	route.listener = nil
}

func (route *TCPRoute) grAcceptConnections() {
	defer route.wg.Done()

	for {
		select {
		case <-route.stopChann:
			return
		default:
			conn, err := route.listener.Accept()
			if err != nil {
				route.PrintError(err)
				continue
			}
			route.connChan <- conn
		}
	}
}

func (route *TCPRoute) grHandleConnections() {
	defer route.wg.Done()

	for {
		select {
		case <-route.stopChann:
			return
		case conn := <-route.connChan:
			route.wg.Add(1)
			go route.grHandleConnection(conn)
		}
	}
}

func (route *TCPRoute) grHandleConnection(clientConn net.Conn) {
	defer clientConn.Close()
	defer route.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), tcpDialTimeout)
	defer cancel()

	serverAddr := fmt.Sprintf("%s:%s", route.TargetHost, route.TargetPort)
	dialer := &net.Dialer{}
	serverConn, err := dialer.DialContext(ctx, route.TargetScheme, serverAddr)
	if err != nil {
		log.Printf("[Stream Dial] %v", err)
		return
	}
	route.tcpPipe(clientConn, serverConn)
}

func (route *TCPRoute) tcpPipe(src net.Conn, dest net.Conn) {
	close := func() {
		src.Close()
		dest.Close()
	}

	var wg sync.WaitGroup
	wg.Add(2) // Number of goroutines

	go func() {
		_, err := io.Copy(src, dest)
		route.PrintError(err)
		close()
		wg.Done()
	}()
	go func() {
		_, err := io.Copy(dest, src)
		route.PrintError(err)
		close()
		wg.Done()
	}()
	wg.Wait()
}
