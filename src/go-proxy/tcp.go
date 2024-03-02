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

func listenTCP(route *StreamRoute) {
	in, err := net.Listen(
		route.ListeningScheme,
		fmt.Sprintf(":%s", route.ListeningPort),
	)
	if err != nil {
		log.Printf("[Stream Listen] %v", err)
		return
	}

	defer in.Close()

	for {
		select {
		case <-route.Context.Done():
			return
		default:
			clientConn, err := in.Accept()
			if err != nil {
				log.Printf("[Stream Accept] %v", err)
				return
			}
			go connectTCPPipe(route, clientConn)
		}
	}
}

func connectTCPPipe(route *StreamRoute, clientConn net.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), tcpDialTimeout)
	defer cancel()

	serverAddr := fmt.Sprintf("%s:%s", route.TargetHost, route.TargetPort)
	dialer := &net.Dialer{}
	serverConn, err := dialer.DialContext(ctx, route.TargetScheme, serverAddr)
	if err != nil {
		log.Printf("[Stream Dial] %v", err)
		return
	}
	tcpPipe(route, clientConn, serverConn)
}

func tcpPipe(route *StreamRoute, src net.Conn, dest net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2) // Number of goroutines
	defer src.Close()
	defer dest.Close()

	go func() {
		_, err := io.Copy(src, dest)
		go route.PrintError(err)
		wg.Done()
	}()
	go func() {
		_, err := io.Copy(dest, src)
		go route.PrintError(err)
		wg.Done()
	}()

	wg.Wait()
}
