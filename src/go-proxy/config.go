package main

import (
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// commented out if unused
type Config interface {
	Value() configModel
	// Load() error
	MustLoad()
	GetAutoCertProvider() (AutoCertProvider, error)
	// MustReload()
	Reload() error
	StartProviders()
	StopProviders()
	WatchChanges()
	StopWatching()
}

func NewConfig(path string) Config {
	cfg := &config{
		reader: &FileReader{Path: path},
	}
	cfg.watcher = NewFileWatcher(
		path,
		cfg.MustReload,        // OnChange
		func() { os.Exit(1) }, // OnDelete
	)
	return cfg
}

func ValidateConfig(data []byte) error {
	cfg := &config{reader: &ByteReader{data}}
	return cfg.Load()
}

func (cfg *config) Value() configModel {
	return *cfg.m
}

func (cfg *config) Load(reader ...Reader) error {
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()

	if cfg.reader == nil {
		panic("config reader not set")
	}

	data, err := cfg.reader.Read()
	if err != nil {
		return NewNestedError("unable to read config file").With(err)
	}

	model := defaultConfig()
	if err := yaml.Unmarshal(data, model); err != nil {
		return NewNestedError("unable to parse config file").With(err)
	}

	ne := NewNestedError("invalid config")

	err = validateYaml(configSchema, data)
	if err != nil {
		ne.With(err)
	}

	pErrs := NewNestedError("errors in these providers")

	for name, p := range model.Providers {
		if p.Kind != ProviderKind_File {
			continue
		}
		_, err := p.ValidateFile()
		if err != nil {
			pErrs.ExtraError(
				NewNestedError("provider file validation error").
					Subject(name).
					With(err),
			)
		}
	}
	if pErrs.HasExtras() {
		ne.With(pErrs)
	}
	if ne.HasInner() {
		return ne
	}

	cfg.m = model
	return nil
}

func (cfg *config) MustLoad() {
	if err := cfg.Load(); err != nil {
		cfgl.Fatal(err)
	}
}

func (cfg *config) GetAutoCertProvider() (AutoCertProvider, error) {
	return cfg.m.AutoCert.GetProvider()
}

func (cfg *config) Reload() error {
	cfg.StopProviders()
	if err := cfg.Load(); err != nil {
		return err
	}
	cfg.StartProviders()
	return nil
}

func (cfg *config) MustReload() {
	if err := cfg.Reload(); err != nil {
		cfgl.Fatal(err)
	}
}

func (cfg *config) StartProviders() {
	if cfg.providerInitialized {
		return
	}

	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()
	if cfg.providerInitialized {
		return
	}

	pErrs := NewNestedError("failed to start these providers")

	ParallelForEachKeyValue(cfg.m.Providers, func(name string, p *Provider) {
		err := p.Init(name)
		if err != nil {
			pErrs.ExtraError(NewNestedErrorFrom(err).Subjectf("%s providers %q", p.Kind, name))
			delete(cfg.m.Providers, name)
		}
		p.StartAllRoutes()
	})

	cfg.providerInitialized = true

	if pErrs.HasExtras() {
		cfgl.Error(pErrs)
	}
}

func (cfg *config) StopProviders() {
	if !cfg.providerInitialized {
		return
	}

	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()
	if !cfg.providerInitialized {
		return
	}
	ParallelForEachValue(cfg.m.Providers, (*Provider).StopAllRoutes)
	cfg.m.Providers = make(map[string]*Provider)
	cfg.providerInitialized = false
}

func (cfg *config) WatchChanges() {
	if cfg.watcher == nil {
		return
	}
	cfg.watcher.Start()
}

func (cfg *config) StopWatching() {
	if cfg.watcher == nil {
		return
	}
	cfg.watcher.Stop()
}

type configModel struct {
	Providers       map[string]*Provider `yaml:",flow" json:"providers"`
	AutoCert        AutoCertConfig       `yaml:",flow" json:"autocert"`
	TimeoutShutdown time.Duration        `yaml:"timeout_shutdown" json:"timeout_shutdown"`
	RedirectToHTTPS bool                 `yaml:"redirect_to_https" json:"redirect_to_https"`
}

func defaultConfig() *configModel {
	return &configModel{
		TimeoutShutdown: 3 * time.Second,
		RedirectToHTTPS: false,
	}
}

type config struct {
	m *configModel

	reader              Reader
	watcher             Watcher
	mutex               sync.Mutex
	providerInitialized bool
}
