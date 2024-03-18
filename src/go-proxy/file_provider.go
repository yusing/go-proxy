package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func (p *Provider) getFileProxyConfigs() ([]*ProxyConfig, error) {
	path := p.Value

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("unable to read config file %q: %v", path, err)
		}
		configMap := make(map[string]ProxyConfig, 0)
		configs := make([]*ProxyConfig, 0)
		err = yaml.Unmarshal(data, &configMap)
		if err != nil {
			return nil, fmt.Errorf("unable to parse config file %q: %v", path, err)
		}

		for alias, cfg := range configMap {
			cfg.Alias = alias
			err = cfg.SetDefaults()
			if err != nil {
				return nil, err
			}
			configs = append(configs, &cfg)
		}
		return configs, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", path)
	} else {
		return nil, err
	}
}
