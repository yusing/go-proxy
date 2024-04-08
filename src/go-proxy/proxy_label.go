package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type ProxyLabel struct {
	Alias string
	Field string
	Value any
}

var errNotProxyLabel = errors.New("not a proxy label")
var errInvalidSetHeaderLine = errors.New("invalid set header line")
var errInvalidBoolean = errors.New("invalid boolean")

const proxyLabelNamespace = "proxy"

func parseProxyLabel(label string, value string) (*ProxyLabel, error) {
	ns := strings.Split(label, ".")
	var v any = value

	if len(ns) != 3 {
		return nil, errNotProxyLabel
	}

	if ns[0] != proxyLabelNamespace {
		return nil, errNotProxyLabel
	}

	field := ns[2]

	var err error
	parser, ok := valueParser[field]

	if ok {
		v, err = parser(v.(string))
		if err != nil {
			return nil, err
		}
	}

	return &ProxyLabel{
		Alias: ns[1],
		Field: field,
		Value: v,
	}, nil
}

func setHeadersParser(value string) (any, error) {
	value = strings.TrimSpace(value)
	lines := strings.Split(value, "\n")
	h := make(http.Header)
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: %q", errInvalidSetHeaderLine, line)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		h.Add(key, val)
	}
	return h, nil
}

func commaSepParser(value string) (any, error) {
	v := strings.Split(value, ",")
	for i := range v {
		v[i] = strings.TrimSpace(v[i])
	}
	return v, nil
}

func boolParser(value string) (any, error) {
	switch strings.ToLower(value) {
	case "true", "yes", "1":
		return true, nil
	case "false", "no", "0":
		return false, nil
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidBoolean, value)
	}
}

var valueParser = map[string]func(string) (any, error){
	"set_headers":   setHeadersParser,
	"hide_headers":  commaSepParser,
	"no_tls_verify": boolParser,
}
