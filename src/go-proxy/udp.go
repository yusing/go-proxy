package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

const udpBufferSize = 1500

// const udpListenTimeout = 100 * time.Second
// const udpConnectionTimeout = 30 * time.Second

type UDPRoute struct {
	*StreamRouteBase

	connMap      map[net.Addr]net.Conn
	connMapMutex sync.Mutex

	listeningConn *net.UDPConn
	targetConn    *net.UDPConn

	connChan chan *UDPConn
}

type UDPConn struct {
	remoteAddr    net.Addr
	buffer        []byte
	bytesReceived []byte
	nReceived     int
}

func NewUDPRoute(config *ProxyConfig) (StreamRoute, error) {
	base, err := newStreamRouteBase(config)
	if err != nil {
		return nil, err
	}

	if base.TargetScheme != UDPStreamType {
		return nil, fmt.Errorf("udp to %s not yet supported", base.TargetScheme)
	}

	return &UDPRoute{
		StreamRouteBase: base,
		connMap:         make(map[net.Addr]net.Conn),
		connChan:        make(chan *UDPConn),
	}, nil
}

func (route *UDPRoute) Listen() {
	source, err := net.ListenPacket(route.ListeningScheme, fmt.Sprintf(":%s", route.ListeningPort))
	if err != nil {
		route.PrintError(err)
		return
	}

	target, err := net.Dial(route.TargetScheme, fmt.Sprintf("%s:%s", route.TargetHost, route.TargetPort))
	if err != nil {
		route.PrintError(err)
		source.Close()
		return
	}

	route.listeningConn = source.(*net.UDPConn)
	route.targetConn = target.(*net.UDPConn)

	route.wg.Add(2)
	go route.grAcceptConnections()
	go route.grHandleConnections()
}

func (route *UDPRoute) StopListening() {
	stopListening(route)
}

func (route *UDPRoute) closeListeners() {
	if route.listeningConn != nil {
		route.listeningConn.Close()
	}
	if route.targetConn != nil {
		route.targetConn.Close()
	}
	route.listeningConn = nil
	route.targetConn = nil
	for _, conn := range route.connMap {
		conn.(*net.UDPConn).Close() // TODO: change on non udp target
	}
}

func (route *UDPRoute) grAcceptConnections() {
	defer route.wg.Done()

	for {
		select {
		case <-route.stopChann:
			return
		default:
			conn, err := route.accept()
			if err != nil {
				route.PrintError(err)
				continue
			}
			route.connChan <- conn
		}
	}
}

func (route *UDPRoute) grHandleConnections() {
	defer route.wg.Done()

	for {
		select {
		case <-route.stopChann:
			return
		case conn := <-route.connChan:
			go func() {
				err := route.handleConnection(conn)
				if err != nil {
					route.PrintError(err)
				}
			}()
		}
	}
}

func (route *UDPRoute) handleConnection(conn *UDPConn) error {
	var err error

	srcConn, ok := route.connMap[conn.remoteAddr]
	if !ok {
		route.connMapMutex.Lock()
		srcConn, err = net.DialUDP("udp", nil, conn.remoteAddr.(*net.UDPAddr))
		if err != nil {
			return err
		}
		route.connMap[conn.remoteAddr] = srcConn
		route.connMapMutex.Unlock()
	}

	// initiate connection to target
	err = route.forwardReceived(conn, route.targetConn)
	if err != nil {
		return err
	}

	for {
		select {
		case <-route.stopChann:
			return nil
		default:
			// receive from target
			conn, err = route.readFrom(route.targetConn, conn.buffer)
			if err != nil {
				return err
			}
			// forward to source
			err = route.forwardReceived(conn, srcConn)
			if err != nil {
				return err
			}
			// read from source
			conn, err = route.readFrom(srcConn, conn.buffer)
			if err != nil {
				continue
			}
			// forward to target
			err = route.forwardReceived(conn, route.targetConn)
			if err != nil {
				return err
			}
		}
	}
}

func (route *UDPRoute) accept() (*UDPConn, error) {
	in := route.listeningConn

	buffer := make([]byte, udpBufferSize)
	nRead, srcAddr, err := in.ReadFromUDP(buffer)

	if err != nil {
		return nil, err
	}

	if nRead == 0 {
		return nil, io.ErrShortBuffer
	}

	return &UDPConn{
			remoteAddr:    srcAddr,
			buffer:        buffer,
			bytesReceived: buffer[:nRead],
			nReceived:     nRead},
		nil
}

func (route *UDPRoute) readFrom(src net.Conn, buffer []byte) (*UDPConn, error) {
	nRead, err := src.Read(buffer)

	if err != nil {
		return nil, err
	}

	if nRead == 0 {
		return nil, io.ErrShortBuffer
	}

	return &UDPConn{
		remoteAddr:    src.RemoteAddr(),
		buffer:        buffer,
		bytesReceived: buffer[:nRead],
		nReceived:     nRead,
	}, nil
}

func (route *UDPRoute) forwardReceived(receivedConn *UDPConn, dest net.Conn) error {
	route.Logf(
		"forwarding %d bytes %s -> %s",
		receivedConn.nReceived,
		receivedConn.remoteAddr.String(),
		dest.RemoteAddr().String(),
	)
	nWritten, err := dest.Write(receivedConn.bytesReceived)

	if nWritten != receivedConn.nReceived {
		err = io.ErrShortWrite
	}

	return err
}
