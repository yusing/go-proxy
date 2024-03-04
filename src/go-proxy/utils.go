package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type Utils struct {
	PortsInUse      map[int]bool
	portsInUseMutex sync.Mutex
}

var utils = &Utils{
	PortsInUse:      make(map[int]bool),
	portsInUseMutex: sync.Mutex{},
}

func (u *Utils) findUseFreePort(startingPort int) (int, error) {
	u.portsInUseMutex.Lock()
	defer u.portsInUseMutex.Unlock()
	for port := startingPort; port <= startingPort+100 && port <= 65535; port++ {
		if u.PortsInUse[port] {
			continue
		}
		addr := fmt.Sprintf(":%d", port)
		l, err := net.Listen("tcp", addr)
		if err == nil {
			u.PortsInUse[port] = true
			l.Close()
			return port, nil
		}
	}
	l, err := net.Listen("tcp", ":0")
	if err == nil {
		// NOTE: may not be after 20000
		port := l.Addr().(*net.TCPAddr).Port
		u.PortsInUse[port] = true
		l.Close()
		return port, nil
	}
	return -1, fmt.Errorf("unable to find free port: %v", err)
}

func (u *Utils) resetPortsInUse() {
	u.portsInUseMutex.Lock()
	for port := range u.PortsInUse {
		u.PortsInUse[port] = false
	}
	u.portsInUseMutex.Unlock()
}

func (u *Utils) markPortInUse(port int) {
	u.portsInUseMutex.Lock()
	u.PortsInUse[port] = true
	u.portsInUseMutex.Unlock()
}

func (*Utils) healthCheckHttp(targetUrl string) error {
	// try HEAD first
	// if HEAD is not allowed, try GET
	resp, err := healthCheckHttpClient.Head(targetUrl)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil && resp != nil && resp.StatusCode == http.StatusMethodNotAllowed {
		_, err = healthCheckHttpClient.Get(targetUrl)
	}
	if resp != nil {
		resp.Body.Close()
	}
	return err
}

func (*Utils) healthCheckStream(scheme string, host string) error {
	conn, err := net.DialTimeout(scheme, host, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
