package docker

import (
	"net/http"
	"strings"

	E "github.com/yusing/go-proxy/error"
)

func setHeadersParser(value string) (any, E.NestedError) {
	value = strings.TrimSpace(value)
	lines := strings.Split(value, "\n")
	h := make(http.Header)
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, E.Invalid("set header statement", line)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		h.Add(key, val)
	}
	return h, E.Nil()
}

func commaSepParser(value string) (any, E.NestedError) {
	v := strings.Split(value, ",")
	for i := range v {
		v[i] = strings.TrimSpace(v[i])
	}
	return v, E.Nil()
}

func boolParser(value string) (any, E.NestedError) {
	switch strings.ToLower(value) {
	case "true", "yes", "1":
		return true, E.Nil()
	case "false", "no", "0":
		return false, E.Nil()
	default:
		return nil, E.Invalid("boolean value", value)
	}
}

const NSProxy = "proxy"

var _ = func() int {
	RegisterNamespace(NSProxy, ValueParserMap{
		"aliases":       commaSepParser,
		"set_headers":   setHeadersParser,
		"hide_headers":  commaSepParser,
		"no_tls_verify": boolParser,
	})
	return 0
}()
