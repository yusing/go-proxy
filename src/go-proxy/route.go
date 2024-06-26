package main

type Route interface {
	Start()
	Stop()
}

func NewRoute(cfg *ProxyConfig) (Route, error) {
	if isStreamScheme(cfg.Scheme) {
		id := cfg.GetID()
		if streamRoutes.Contains(id) {
			return nil, NewNestedError("duplicated stream").Subject(cfg.Alias)
		}
		route, err := NewStreamRoute(cfg)
		if err != nil {
			return nil, NewNestedErrorFrom(err).Subject(cfg.Alias)
		}
		return route, nil
	} else {
		httpRoutes.Ensure(cfg.Alias)
		route, err := NewHTTPRoute(cfg)
		if err != nil {
			return nil, NewNestedErrorFrom(err).Subject(cfg.Alias)
		}
		return route, nil
	}
}

func isValidScheme(s string) bool {
	for _, v := range ValidSchemes {
		if v == s {
			return true
		}
	}
	return false
}

func isStreamScheme(s string) bool {
	for _, v := range StreamSchemes {
		if v == s {
			return true
		}
	}
	return false
}
