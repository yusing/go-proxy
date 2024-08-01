package fields

import (
	"strings"

	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
)

type StreamPort struct {
	ListeningPort Port `json:"listening"`
	ProxyPort     Port `json:"proxy"`
}

func NewStreamPort(p string) (StreamPort, E.NestedError) {
	split := strings.Split(p, ":")
	if len(split) != 2 {
		return StreamPort{}, E.Invalid("stream port", p).Extra("should be in 'x:y' format")
	}

	listeningPort, err := NewPort(split[0])
	if err.IsNotNil() {
		return StreamPort{}, err
	}
	if err = listeningPort.boundCheck(); err.IsNotNil() {
		return StreamPort{}, err
	}

	proxyPort, err := NewPort(split[1])
	if err.IsNotNil() {
		proxyPort, err = parseNameToPort(split[1])
		if err.IsNotNil() {
			return StreamPort{}, err
		}
	}
	if err = proxyPort.boundCheck(); err.IsNotNil() {
		return StreamPort{}, err
	}

	return StreamPort{ListeningPort: listeningPort, ProxyPort: proxyPort}, E.Nil()
}

func parseNameToPort(name string) (Port, E.NestedError) {
	port, ok := common.NamePortMapTCP[name]
	if !ok {
		return -1, E.Unsupported("service", name)
	}
	return Port(port), E.Nil()
}
