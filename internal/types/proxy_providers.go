package types

type ProxyProviders struct {
	Files  []string          `json:"include" yaml:"include"` // docker, file
	Docker map[string]string `json:"docker" yaml:"docker"`
}
