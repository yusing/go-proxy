package agent

import (
	"net"
	"strings"
)

func MachineIP() (string, bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		interfaces = []net.Interface{}
	}
	for _, in := range interfaces {
		addrs, err := in.Addrs()
		if err != nil {
			continue
		}
		if !strings.HasPrefix(in.Name, "eth") && !strings.HasPrefix(in.Name, "en") {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String(), true
				}
			}
		}
	}
	return "", false
}
