package main

import (
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

func (p *Provider) GetFilePath() string {
	return path.Join(configBasePath, p.Value)
}

func (p *Provider) ValidateFile() (ProxyConfigSlice, error) {
	path := p.GetFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewNestedError("unable to read providers file").Subject(path).With(err)
	}
	result, err := ValidateFileContent(data)
	if err != nil {
		return nil, NewNestedError(err.Error()).Subject(path)
	}
	return result, nil
}

func ValidateFileContent(data []byte) (ProxyConfigSlice, error) {
	configMap := make(ProxyConfigMap, 0)
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return nil, NewNestedError("invalid yaml").With(err)
	}

	ne := NewNestedError("errors in providers")

	configs := make(ProxyConfigSlice, len(configMap))
	i := 0
	for alias, cfg := range configMap {
		cfg.Alias = alias
		if err := cfg.SetDefaults(); err != nil {
			ne.ExtraError(err)
		} else {
			configs[i] = cfg
		}
		i++
	}

	if err := validateYaml(providersSchema, data); err != nil {
		ne.ExtraError(err)
	}
	if ne.HasExtras() {
		return nil, ne
	}
	return configs, nil
}
