package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type Utils struct {
	PortsInUse map[int]bool
	portsInUseMutex sync.Mutex
}
var utils = &Utils{
	PortsInUse: make(map[int]bool),
	portsInUseMutex: sync.Mutex{},
}

func (u *Utils) findFreePort(startingPort int) (int, error) {
	for port := startingPort; port <= startingPort+100 && port <= 65535; port++ {
		if u.PortsInUse[port] {
			continue
		}
		addr := fmt.Sprintf(":%d", port)
		l, err := net.Listen("tcp", addr)
		if err == nil {
			l.Close()
			return port, nil
		}
	}
	l, err := net.Listen("tcp", ":0")
	if err == nil {
		l.Close()
		// NOTE: may not be after 20000
		return l.Addr().(*net.TCPAddr).Port, nil
	}
	return -1, fmt.Errorf("unable to find free port: %v", err)
}

func (u *Utils) resetPortsInUse() {
	u.portsInUseMutex.Lock()
	defer u.portsInUseMutex.Unlock()
	for port := range u.PortsInUse {
		u.PortsInUse[port] = false
	}
}

func (u* Utils) markPortInUse(port int) {
	u.portsInUseMutex.Lock()
	defer u.portsInUseMutex.Unlock()
	u.PortsInUse[port] = true
}

func (*Utils) healthCheckHttp(targetUrl string) error {
	// try HEAD first
	// if HEAD is not allowed, try GET
	resp, err := healthCheckHttpClient.Head(targetUrl)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil && resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
		_, err = healthCheckHttpClient.Get(targetUrl)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}

func (*Utils) healthCheckStream(scheme string, host string) error {
	conn, err := net.DialTimeout(scheme, host, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
