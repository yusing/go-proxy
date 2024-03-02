package main

import (
	"net"
	"net/http"
	"time"
)

func healthCheckHttp(targetUrl string) error {
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

func healthCheckStream(scheme string, host string) error {
	conn, err := net.DialTimeout(scheme, host, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
