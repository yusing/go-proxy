package loadbalancer

import (
	U "github.com/yusing/go-proxy/internal/utils"
)

type Mode string

const (
	RoundRobin Mode = "roundrobin"
	LeastConn  Mode = "leastconn"
	IPHash     Mode = "iphash"
)

func (mode *Mode) ValidateUpdate() bool {
	switch U.ToLowerNoSnake(string(*mode)) {
	case "", string(RoundRobin):
		*mode = RoundRobin
		return true
	case string(LeastConn):
		*mode = LeastConn
		return true
	case string(IPHash):
		*mode = IPHash
		return true
	}
	*mode = RoundRobin
	return false
}
