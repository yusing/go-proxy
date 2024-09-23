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

func ValidateStreamPort(p string) (StreamPort, E.NestedError) {
	split := strings.Split(p, ":")

	switch len(split) {
	case 1:
		split = []string{"0", split[0]}
	case 2:
		break
	default:
		return ErrStreamPort, E.Invalid("stream port", p).With("too many colons")
	}

	listeningPort, err := ValidatePort(split[0])
	if err != nil {
		return ErrStreamPort, err
	}

	proxyPort, err := ValidatePort(split[1])
	if err.Is(E.ErrOutOfRange) {
		return ErrStreamPort, err
	} else if proxyPort == 0 {
		return ErrStreamPort, E.Invalid("stream port", p).With("proxy port cannot be 0")
	} else if err != nil {
		proxyPort, err = parseNameToPort(split[1])
		if err != nil {
			return ErrStreamPort, E.Invalid("stream port", p).With(proxyPort)
		}
	}

	return StreamPort{listeningPort, proxyPort}, nil
}

func parseNameToPort(name string) (Port, E.NestedError) {
	port, ok := common.ServiceNamePortMapTCP[name]
	if !ok {
		return ErrPort, E.Invalid("service", name)
	}
	return Port(port), nil
}

var ErrStreamPort = StreamPort{ErrPort, ErrPort}
