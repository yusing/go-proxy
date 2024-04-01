package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
)

type UDPRoute struct {
	*StreamRouteBase

	connMap      UDPConnMap
	connMapMutex sync.Mutex

	listeningConn *net.UDPConn
	targetAddr    *net.UDPAddr
}

type UDPConn struct {
	src *net.UDPConn
	dst *net.UDPConn
	*BidirectionalPipe
}

type UDPConnMap map[string]*UDPConn

func NewUDPRoute(base *StreamRouteBase) StreamImpl {
	return &UDPRoute{
		StreamRouteBase: base,
		connMap:         make(UDPConnMap),
	}
}

func (route *UDPRoute) Setup() error {
	laddr, err := net.ResolveUDPAddr(route.ListeningScheme, fmt.Sprintf(":%v", route.ListeningPort))
	if err != nil {
		return err
	}
	source, err := net.ListenUDP(route.ListeningScheme, laddr)
	if err != nil {
		return err
	}
	raddr, err := net.ResolveUDPAddr(route.TargetScheme, fmt.Sprintf("%s:%v", route.TargetHost, route.TargetPort))
	if err != nil {
		source.Close()
		return err
	}

	route.listeningConn = source
	route.targetAddr = raddr
	return nil
}

func (route *UDPRoute) Accept() (interface{}, error) {
	in := route.listeningConn

	buffer := make([]byte, udpBufferSize)
	nRead, srcAddr, err := in.ReadFromUDP(buffer)

	if err != nil {
		return nil, err
	}

	if nRead == 0 {
		return nil, io.ErrShortBuffer
	}

	key := srcAddr.String()
	conn, ok := route.connMap[key]

	if !ok {
		route.connMapMutex.Lock()
		if conn, ok = route.connMap[key]; !ok {
			srcConn, err := net.DialUDP("udp", nil, srcAddr)
			if err != nil {
				return nil, err
			}
			dstConn, err := net.DialUDP("udp", nil, route.targetAddr)
			if err != nil {
				srcConn.Close()
				return nil, err
			}
			pipeCtx, pipeCancel := context.WithCancel(context.Background())
			go func() {
				<-route.stopCh
				pipeCancel()
			}()
			conn = &UDPConn{
				srcConn,
				dstConn,
				NewBidirectionalPipe(pipeCtx, sourceRWCloser{in, dstConn}, sourceRWCloser{in, srcConn}),
			}
			route.connMap[key] = conn
		}
		route.connMapMutex.Unlock()
	}

	_, err = conn.dst.Write(buffer[:nRead])
	return conn, err
}

func (route *UDPRoute) Handle(c interface{}) error {
	return c.(*UDPConn).Start()
}

func (route *UDPRoute) CloseListeners() {
	if route.listeningConn != nil {
		route.listeningConn.Close()
		route.listeningConn = nil
	}
	for _, conn := range route.connMap {
		if err := conn.src.Close(); err != nil {
			route.l.Errorf("error closing src conn: %w", err)
		}
		if err := conn.dst.Close(); err != nil {
			route.l.Error("error closing dst conn: %w", err)
		}
	}
	route.connMap = make(UDPConnMap)
}

type sourceRWCloser struct {
	server *net.UDPConn
	*net.UDPConn
}

func (w sourceRWCloser) Write(p []byte) (int, error) {
	return w.server.WriteToUDP(p, w.RemoteAddr().(*net.UDPAddr)) // TODO: support non udp
}
