package main

import (
	"encoding/base64"
	"log"
	"net"
	"os"

	"github.com/yusing/go-proxy/agent/pkg/certs"
)

func NewAgent(args []string) {
	if len(args) != 2 {
		log.Fatalf("invalid arguments: %v", args)
	}
	host := args[0]
	certDataBase64 := args[1]

	ip, _, err := net.SplitHostPort(host)
	if err != nil {
		log.Fatalf("invalid host: %v", err)
	}

	_, err = net.ResolveIPAddr("ip", ip)
	if err != nil {
		log.Fatalf("invalid host: %v", err)
	}

	certData, err := base64.StdEncoding.DecodeString(certDataBase64)
	if err != nil {
		log.Fatalf("invalid cert data: %v", err)
	}

	f, err := os.OpenFile(certs.AgentCertsFilename(host), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("failed to create file: %v", err)
	}
	defer f.Close()

	_, err = f.Write(certData)
	if err != nil {
		log.Fatalf("failed to write cert data: %v", err)
	}

	log.Printf("agent cert created: %s", certs.AgentCertsFilename(host))
}
