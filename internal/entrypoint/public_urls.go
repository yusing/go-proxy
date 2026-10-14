package entrypoint

import (
	"net"
	"strings"

	"github.com/yusing/godoxy/internal/routing"
)

// PublicURLs derives public HTTP URLs from this runtime's routing domains and
// listener configuration, never from an API request or an upstream address.
func (ep *Entrypoint) PublicURLs(r routing.Route) []string {
	httpRoute, ok := r.(routing.HTTPRoute)
	if !ok || r.ShouldExclude() {
		return nil
	}

	httpAddr, httpsAddr := getAddr(httpRoute)
	scheme, addr, defaultPort := "http", httpAddr, "80"
	if httpsAddr != "" {
		scheme, addr, defaultPort = "https", httpsAddr, "443"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}
	publicURL := func(host string) string {
		if port != defaultPort {
			host = net.JoinHostPort(host, port)
		}
		return scheme + "://" + host
	}

	alias := r.Name()
	if len(ep.matchDomains) == 0 {
		if strings.Contains(alias, ".") {
			return []string{publicURL(alias)}
		}
		return nil
	}
	for _, domain := range ep.matchDomains {
		domain = strings.TrimPrefix(domain, ".")
		if alias == domain || strings.HasSuffix(alias, "."+domain) {
			return []string{publicURL(alias)}
		}
	}
	urls := make([]string, 0, len(ep.matchDomains))
	for _, domain := range ep.matchDomains {
		urls = append(urls, publicURL(alias+"."+strings.TrimPrefix(domain, ".")))
	}
	return urls
}
