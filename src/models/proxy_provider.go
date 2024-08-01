package model

type (
	ProxyProvider struct {
		Kind  string `json:"kind"` // docker, file
		Value string `json:"value"`
	}
	ProxyProviders = map[string]ProxyProvider
)
