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
	cfg.StopProviders()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("unable to read config file: %v", err)
	}

	cfg.Providers = make(map[string]*Provider)
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("unable to parse config file: %v", err)
	}

	for name, p := range cfg.Providers {
		err := p.Init(name)
		if err != nil {
			cfgl.Errorf("failed to initialize provider %q %v", name, err)
			cfg.Providers[name] = nil
		}
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
	if cfg.Providers == nil {
		cfgl.Fatal("providers not loaded")
	}
	// Providers have their own mutex, no lock needed
	ParallelForEachValue(cfg.Providers, (*Provider).StartAllRoutes)
}

func (cfg *config) StopProviders() {
	if cfg.Providers != nil {
		// Providers have their own mutex, no lock needed
		ParallelForEachValue(cfg.Providers, (*Provider).StopAllRoutes)
	}
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
