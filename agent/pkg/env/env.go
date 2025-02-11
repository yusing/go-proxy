package env

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/yusing/go-proxy/internal/common"
)

func DefaultAgentName() string {
	name, err := os.Hostname()
	if err != nil {
		return "agent"
	}
	return name
}

var (
	AgentName                = common.GetEnvString("AGENT_NAME", DefaultAgentName())
	AgentPort                = common.GetEnvInt("AGENT_PORT", 8890)
	AgentRegistrationPort    = common.GetEnvInt("AGENT_REGISTRATION_PORT", 8891)
	AgentSkipClientCertCheck = common.GetEnvBool("AGENT_SKIP_CLIENT_CERT_CHECK", false)

	RegistrationAllowedHosts = common.GetCommaSepEnv("REGISTRATION_ALLOWED_HOSTS", "")
	RegistrationAllowedCIDRs []*net.IPNet
)

func init() {
	cidrs, err := toCIDRs(RegistrationAllowedHosts)
	if err != nil {
		log.Fatalf("failed to parse allowed hosts: %v", err)
	}
	RegistrationAllowedCIDRs = cidrs
}

func toCIDRs(hosts []string) ([]*net.IPNet, error) {
	cidrs := make([]*net.IPNet, 0, len(hosts))
	for _, host := range hosts {
		if !strings.Contains(host, "/") {
			host += "/32"
		}
		_, cidr, err := net.ParseCIDR(host)
		if err != nil {
			return nil, err
		}
		cidrs = append(cidrs, cidr)
	}
	return cidrs, nil
}

var warnOnce sync.Once

func IsAllowedHost(remoteAddr string) bool {
	if len(RegistrationAllowedCIDRs) == 0 {
		warnOnce.Do(func() {
			log.Println("Warning: REGISTRATION_ALLOWED_HOSTS is empty, allowing all hosts")
		})
		return true
	}
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return false
	}
	for _, cidr := range RegistrationAllowedCIDRs {
		if cidr.Contains(netIP) {
			return true
		}
	}
	return false
}
