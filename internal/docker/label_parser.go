package docker

import (
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
	"gopkg.in/yaml.v3"
)

const (
	NSProxy                    = "proxy"
	ProxyAttributePathPatterns = "path_patterns"
	ProxyAttributeNoTLSVerify  = "no_tls_verify"
	ProxyAttributeMiddlewares  = "middlewares"
)

var _ = func() int {
	RegisterNamespace(NSProxy, ValueParserMap{
		ProxyAttributePathPatterns: YamlStringListParser,
		ProxyAttributeNoTLSVerify:  BoolParser,
	})
	return 0
}()

func YamlStringListParser(value string) (any, E.NestedError) {
	/*
		- foo
		- bar
		- baz
	*/
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}, nil
	}
	var data []string
	err := E.From(yaml.Unmarshal([]byte(value), &data))
	return data, err
}

func YamlLikeMappingParser(allowDuplicate bool) func(string) (any, E.NestedError) {
	return func(value string) (any, E.NestedError) {
		/*
			foo: bar
			boo: baz
		*/
		value = strings.TrimSpace(value)
		lines := strings.Split(value, "\n")
		h := make(map[string]string)
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return nil, E.Invalid("syntax", line).With("too many colons")
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if existing, ok := h[key]; ok {
				if !allowDuplicate {
					return nil, E.Duplicated("key", key)
				}
				h[key] = existing + ", " + val
			} else {
				h[key] = val
			}
		}
		return h, nil
	}
}

func BoolParser(value string) (any, E.NestedError) {
	switch strings.ToLower(value) {
	case "true", "yes", "1":
		return true, nil
	case "false", "no", "0":
		return false, nil
	default:
		return nil, E.Invalid("boolean value", value)
	}
}
