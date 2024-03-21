package main

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

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

	if base.TargetScheme != StreamType_UDP {
		return nil, fmt.Errorf("udp to %s not yet supported", base.TargetScheme)
	}

	return &UDPRoute{
		StreamRouteBase: base,
		connMap:         make(map[net.Addr]net.Conn),
		connChan:        make(chan *UDPConn),
	}, nil
}

func (route *UDPRoute) Start() {
	route.setupListen()

	source, err := net.ListenPacket(route.ListeningScheme, fmt.Sprintf(":%v", route.ListeningPort))
	if err != nil {
		route.l.Error(err)
		return
	}

	target, err := net.Dial(route.TargetScheme, fmt.Sprintf("%s:%v", route.TargetHost, route.TargetPort))
	if err != nil {
		route.l.Error(err)
		source.Close()
		return
	}

	route.listeningConn = source.(*net.UDPConn)
	route.targetConn = target.(*net.UDPConn)

	route.wg.Add(2)
	go route.grAcceptConnections()
	go route.grHandleConnections()
}

func (route *UDPRoute) Stop() {
	stopListening(route)
	streamRoutes.Delete(route.id)
}

func (route *UDPRoute) closeListeners() {
	if route.listeningConn != nil {
		route.listeningConn.Close()
		route.listeningConn = nil
	}
	if route.targetConn != nil {
		route.targetConn.Close()
		route.targetConn = nil
	}
	for _, conn := range route.connMap {
		conn.(*net.UDPConn).Close() // TODO: change on non udp target
	}
	route.connMap = make(map[net.Addr]net.Conn)
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
				route.l.Error(err)
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
					route.l.Error(err)
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

	var forwarder func(*UDPConn, net.Conn) error

	if logLevel == logrus.DebugLevel {
		forwarder = route.forwardReceivedDebug
	} else {
		forwarder = route.forwardReceivedReal
	}

	// initiate connection to target
	err = forwarder(conn, route.targetConn)
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
			err = forwarder(conn, srcConn)
			if err != nil {
				return err
			}
			// read from source
			conn, err = route.readFrom(srcConn, conn.buffer)
			if err != nil {
				continue
			}
			// forward to target
			err = forwarder(conn, route.targetConn)
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

func (route *UDPRoute) forwardReceivedReal(receivedConn *UDPConn, dest net.Conn) error {
	nWritten, err := dest.Write(receivedConn.bytesReceived)

	if nWritten != receivedConn.nReceived {
		err = io.ErrShortWrite
	}

	return err
}

func (route *UDPRoute) forwardReceivedDebug(receivedConn *UDPConn, dest net.Conn) error {
	route.l.WithField("size", receivedConn.nReceived).Debugf(
		"forwarding from %s to %s",
		receivedConn.remoteAddr.String(),
		dest.RemoteAddr().String(),
	)
	return route.forwardReceivedReal(receivedConn, dest)
}
