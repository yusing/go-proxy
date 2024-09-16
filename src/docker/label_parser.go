package docker

import (
	"strings"

	E "github.com/yusing/go-proxy/error"
	"gopkg.in/yaml.v3"
)

func yamlListParser(value string) (any, E.NestedError) {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}, E.Nil()
	}
	var data []string
	err := E.From(yaml.Unmarshal([]byte(value), &data))
	return data, err
}

func yamlStringMappingParser(value string) (any, E.NestedError) {
	value = strings.TrimSpace(value)
	lines := strings.Split(value, "\n")
	h := make(map[string]string)
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, E.Invalid("set header statement", line)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if existing, ok := h[key]; ok {
			h[key] = existing + ", " + val
		} else {
			h[key] = val
		}
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
		"path_patterns": yamlListParser,
		"set_headers":   yamlStringMappingParser,
		"hide_headers":  yamlListParser,
		"no_tls_verify": boolParser,
	})
	return 0
}()
