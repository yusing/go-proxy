package types

import (
	"net"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type CIDR net.IPNet

var (
	ErrInvalidCIDR     = E.New("invalid CIDR")
	ErrInvalidCIDRType = E.New("invalid CIDR type")
)

func (cidr *CIDR) ConvertFrom(val any) E.Error {
	cidrStr, ok := val.(string)
	if !ok {
		return ErrInvalidCIDRType.Subjectf("%T", val)
	}

	if !strings.Contains(cidrStr, "/") {
		cidrStr += "/32" // single IP
	}
	_, ipnet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return ErrInvalidCIDR.Subject(cidrStr)
	}
	*cidr = CIDR(*ipnet)
	return nil
}

func (cidr *CIDR) Contains(ip net.IP) bool {
	return (*net.IPNet)(cidr).Contains(ip)
}

func (cidr *CIDR) String() string {
	return (*net.IPNet)(cidr).String()
}

func (cidr *CIDR) Equals(other *CIDR) bool {
	return (*net.IPNet)(cidr).IP.Equal(other.IP) && cidr.Mask.String() == other.Mask.String()
}
