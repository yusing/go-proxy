package fields

import (
	"strings"

	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
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
		return ErrStreamPort, err.Subject("listening port")
	}

	proxyPort, err := ValidatePort(split[1])

	if err.Is(E.ErrOutOfRange) {
		return ErrStreamPort, err.Subject("proxy port")
	} else if proxyPort == 0 {
		return ErrStreamPort, E.Invalid("proxy port", p)
	} else if err != nil {
		proxyPort, err = parseNameToPort(split[1])
		if err != nil {
			return ErrStreamPort, E.Invalid("proxy port", proxyPort)
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
