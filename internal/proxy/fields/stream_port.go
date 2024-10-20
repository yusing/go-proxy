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

func ValidateStreamPort(p string) (_ StreamPort, err E.Error) {
	split := strings.Split(p, ":")

	switch len(split) {
	case 1:
		split = []string{"0", split[0]}
	case 2:
		break
	default:
		err = E.Invalid("stream port", p).With("too many colons")
		return
	}

	listeningPort, err := ValidatePort(split[0])
	if err != nil {
		err = err.Subject("listening port")
		return
	}

	proxyPort, err := ValidatePort(split[1])

	if err.Is(E.ErrOutOfRange) {
		err = err.Subject("proxy port")
		return
	} else if err != nil {
		proxyPort, err = parseNameToPort(split[1])
		if err != nil {
			err = E.Invalid("proxy port", proxyPort)
			return
		}
	}

	return StreamPort{listeningPort, proxyPort}, nil
}

func parseNameToPort(name string) (Port, E.Error) {
	port, ok := common.ServiceNamePortMapTCP[name]
	if !ok {
		return ErrPort, E.Invalid("service", name)
	}
	return Port(port), nil
}
