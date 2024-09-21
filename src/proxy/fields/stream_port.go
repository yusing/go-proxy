package fields

import (
	"fmt"
	"strings"

	"github.com/yusing/go-proxy/common"
	E "github.com/yusing/go-proxy/error"
)

type StreamPort struct {
	ListeningPort Port `json:"listening"`
	ProxyPort     Port `json:"proxy"`
}

func ValidateStreamPort(p string) (StreamPort, E.NestedError) {
	split := strings.Split(p, ":")
	if len(split) != 2 {
		return StreamPort{}, E.Invalid("stream port", fmt.Sprintf("%q", p)).With("should be in 'x:y' format")
	}

	listeningPort, err := ValidatePort(split[0])
	if err.HasError() {
		return StreamPort{}, err
	}
	if err = listeningPort.boundCheck(); err.HasError() {
		return StreamPort{}, err
	}

	proxyPort, err := ValidatePort(split[1])
	if err.HasError() {
		proxyPort, err = parseNameToPort(split[1])
		if err.HasError() {
			return StreamPort{}, err
		}
	}
	if err = proxyPort.boundCheck(); err.HasError() {
		return StreamPort{}, err
	}

	return StreamPort{ListeningPort: listeningPort, ProxyPort: proxyPort}, nil
}

func parseNameToPort(name string) (Port, E.NestedError) {
	port, ok := common.ServiceNamePortMapTCP[name]
	if !ok {
		return -1, E.Unsupported("service", name)
	}
	return Port(port), nil
}
