package main

import (
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const udpBufferSize = 1500
const udpMaxQueueSizePerStream = 100
const udpListenTimeout = 100 * time.Second
const udpConnectionTimeout = 30 * time.Second

type UDPRoute struct {
	StreamRoute

	ConnMap       map[net.Addr]*net.UDPConn
	ConnMapMutex  sync.Mutex
	QueueSize     atomic.Int32
	SourceUDPAddr *net.UDPAddr
	TargetUDPAddr *net.UDPAddr
}

func listenUDP(route *UDPRoute) {
	source, err := net.ListenUDP(route.ListeningScheme, route.SourceUDPAddr)
	if err != nil {
		route.PrintError(err)
		return
	}

	target, err := net.DialUDP(route.TargetScheme, nil, route.TargetUDPAddr)
	if err != nil {
		route.PrintError(err)
		return
	}
	var wg sync.WaitGroup

	defer wg.Wait()
	defer source.Close()
	defer target.Close()

	var udpBuffers = [udpMaxQueueSizePerStream][udpBufferSize]byte{}

	for {
		select {
		case <-route.Context.Done():
			return
		default:
			if route.QueueSize.Load() >= udpMaxQueueSizePerStream {
				wg.Wait()
			}
			go udpLoop(
				route,
				source,
				target,
				udpBuffers[route.QueueSize.Load()][:],
				&wg,
			)
		}
	}
}

func udpLoop(route *UDPRoute, in *net.UDPConn, out *net.UDPConn, buffer []byte, wg *sync.WaitGroup) {
	wg.Add(1)
	route.QueueSize.Add(1)
	defer route.QueueSize.Add(-1)
	defer wg.Done()

	var nRead int
	var nWritten int

	in.SetReadDeadline(time.Now().Add(udpListenTimeout))
	nRead, srcAddr, err := in.ReadFromUDP(buffer)

	if err != nil {
		return
	}

	log.Printf("[Stream] received %d bytes from %s, forwarding to %s", nRead, srcAddr.String(), out.RemoteAddr().String())
	out.SetWriteDeadline(time.Now().Add(udpConnectionTimeout))
	nWritten, err = out.Write(buffer[:nRead])
	if nWritten != nRead {
		err = io.ErrShortWrite
	}
	if err != nil {
		go route.PrintError(err)
		return
	}

	err = udpPipe(route, out, srcAddr, buffer)
	if err != nil {
		go route.PrintError(err)
	}
}

func udpPipe(route *UDPRoute, src *net.UDPConn, destAddr *net.UDPAddr, buffer []byte) error {
	src.SetReadDeadline(time.Now().Add(udpConnectionTimeout))
	nRead, err := src.Read(buffer)
	if err != nil || nRead == 0 {
		return err
	}
	log.Printf("[Stream] received %d bytes from %s, forwarding to %s", nRead, src.RemoteAddr().String(), destAddr.String())
	dest, ok := route.ConnMap[destAddr]
	if !ok {
		dest, err = net.DialUDP(src.LocalAddr().Network(), nil, destAddr)
		if err != nil {
			return err
		}
		route.ConnMapMutex.Lock()
		route.ConnMap[destAddr] = dest
		route.ConnMapMutex.Unlock()
	}
	dest.SetWriteDeadline(time.Now().Add(udpConnectionTimeout))
	nWritten, err := dest.Write(buffer[:nRead])
	if err != nil {
		return err
	}
	if nWritten != nRead {
		return io.ErrShortWrite
	}
	return nil
}
