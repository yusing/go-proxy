package types

import (
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type StreamPort struct {
	ListeningPort Port `json:"listening"`
	ProxyPort     Port `json:"proxy"`
}

var ErrStreamPortTooManyColons = E.New("too many colons")

func ValidateStreamPort(p string) (StreamPort, error) {
	split := strutils.SplitRune(p, ':')

	switch len(split) {
	case 1:
		split = []string{"0", split[0]}
	case 2:
		break
	default:
		return StreamPort{}, ErrStreamPortTooManyColons.Subject(p)
	}

	listeningPort, lErr := ValidatePort(split[0])
	proxyPort, pErr := ValidatePort(split[1])
	if err := E.Join(lErr, pErr); err != nil {
		return StreamPort{}, err
	}

	return StreamPort{listeningPort, proxyPort}, nil
}
