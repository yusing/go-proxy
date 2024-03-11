package main

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
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

func (p *Provider) grWatchFileChanges() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		p.Errorf("Watcher", "unable to create file watcher: %v", err)
	}
	defer watcher.Close()

	if err = watcher.Add(p.Value); err != nil {
		p.Errorf("Watcher", "unable to watch file %q: %v", p.Value, err)
		return
	}

	for {
		select {
		case <-p.stopWatching:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			switch {
			case event.Has(fsnotify.Write):
				p.Logf("Watcher", "file change detected", p.name)
				p.StopAllRoutes()
				p.BuildStartRoutes()
			case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
				p.Logf("Watcher", "file renamed / deleted", p.name)
				p.StopAllRoutes()
			}
		case err := <-watcher.Errors:
			p.Errorf("Watcher", "File watcher error: %s", p.name, err)
		}
	}
}
