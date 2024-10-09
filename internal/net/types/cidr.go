package types

import (
	"net"
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type CIDR net.IPNet

func (*CIDR) ConvertFrom(val any) (any, E.NestedError) {
	cidr, ok := val.(string)
	if !ok {
		return nil, E.TypeMismatch[string](val)
	}

	if !strings.Contains(cidr, "/") {
		cidr += "/32" // single IP
	}
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, E.Invalid("CIDR", cidr)
	}
	return (*CIDR)(ipnet), nil
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
