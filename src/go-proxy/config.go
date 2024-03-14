package main

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Providers map[string]*Provider `yaml:",flow"`
}

var config *Config

func ReadConfig() (*Config, error) {
	config := Config{}
	data, err := os.ReadFile(configPath)

	if err != nil {
		return nil, fmt.Errorf("unable to read config file: %v", err)
	}

	err = yaml.Unmarshal(data, &config)

	if err != nil {
		return nil, fmt.Errorf("unable to parse config file: %v", err)
	}

	for name, p := range config.Providers {
		p.name = name
	}

	return &config, nil
}

func ListenConfigChanges() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		glog.Errorf("[Config] unable to create file watcher: %v", err)
	}
	defer watcher.Close()

	if err = watcher.Add(configPath); err != nil {
		glog.Errorf("[Config] unable to watch file: %v", err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			switch {
			case event.Has(fsnotify.Write):
				glog.Infof("[Config] file change detected")
				for _, p := range config.Providers {
					p.StopAllRoutes()
				}
				config, err = ReadConfig()
				if err != nil {
					glog.Fatalf("[Config] unable to read config: %v", err)
				}
				StartAllRoutes()
			case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
				glog.Fatalf("[Config] file renamed / deleted")
			}
		case err := <-watcher.Errors:
			glog.Errorf("[Config] File watcher error: %s", err)
		}
	}
}