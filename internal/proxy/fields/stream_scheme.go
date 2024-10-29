package fields

import (
	"fmt"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type StreamScheme struct {
	ListeningScheme Scheme `json:"listening"`
	ProxyScheme     Scheme `json:"proxy"`
}

func ValidateStreamScheme(s string) (*StreamScheme, error) {
	ss := &StreamScheme{}
	parts := strings.Split(s, ":")
	if len(parts) == 1 {
		parts = []string{s, s}
	} else if len(parts) != 2 {
		return nil, ErrInvalidScheme.Subject(s)
	}

	var lErr, pErr error
	ss.ListeningScheme, lErr = NewScheme(parts[0])
	ss.ProxyScheme, pErr = NewScheme(parts[1])

	if err := E.Join(lErr, pErr); err != nil {
		return nil, err
	}

	return ss, nil
}

func (s StreamScheme) String() string {
	return fmt.Sprintf("%s -> %s", s.ListeningScheme, s.ProxyScheme)
}

// IsCoherent checks if the ListeningScheme and ProxyScheme of the StreamScheme are equal.
//
// It returns a boolean value indicating whether the ListeningScheme and ProxyScheme are equal.
func (s StreamScheme) IsCoherent() bool {
	return s.ListeningScheme == s.ProxyScheme
}
