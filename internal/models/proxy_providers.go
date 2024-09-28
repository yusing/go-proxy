package model

type ProxyProviders struct {
	Files  []string          `yaml:"include" json:"include"` // docker, file
	Docker map[string]string `yaml:"docker" json:"docker"`
}
