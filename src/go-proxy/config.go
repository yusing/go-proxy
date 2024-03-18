package main

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// commented out if unused
type Config interface {
	// Load() error
	MustLoad()
	// MustReload()
	// Reload() error
	StartProviders()
	StopProviders()
	WatchChanges()
	StopWatching()
}

func NewConfig() Config {
	cfg := &config{}
	cfg.watcher = NewFileWatcher(
		configPath,
		cfg.MustReload,        // OnChange
		func() { os.Exit(1) }, // OnDelete
	)
	return cfg
}

func (cfg *config) Load() error {
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()

	// unload if any
	if cfg.Providers != nil {
		for _, p := range cfg.Providers {
			p.StopAllRoutes()
		}
	}
	cfg.Providers = make(map[string]*Provider)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("unable to read config file: %v", err)
	}

	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("unable to parse config file: %v", err)
	}

	for name, p := range cfg.Providers {
		p.name = name
	}

	return nil
}

func (cfg *config) MustLoad() {
	if err := cfg.Load(); err != nil {
		cfgl.Fatal(err)
	}
}

func (cfg *config) Reload() error {
	return cfg.Load()
}

func (cfg *config) MustReload() {
	cfg.MustLoad()
}

func (cfg *config) StartProviders() {
	// Providers have their own mutex, no lock needed
	ParallelForEachValue(cfg.Providers, (*Provider).StartAllRoutes)
}

func (cfg *config) StopProviders() {
	// Providers have their own mutex, no lock needed
	ParallelForEachValue(cfg.Providers, (*Provider).StopAllRoutes)
}

func (cfg *config) WatchChanges() {
	cfg.watcher.Start()
}

func (cfg *config) StopWatching() {
	cfg.watcher.Stop()
}

type config struct {
	Providers map[string]*Provider `yaml:",flow"`
	watcher   Watcher
	mutex     sync.Mutex
}
