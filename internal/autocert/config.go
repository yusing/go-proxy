package autocert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	E "github.com/yusing/go-proxy/internal/error"

	"github.com/yusing/go-proxy/internal/config/types"
)

type Config types.AutoCertConfig

func NewConfig(cfg *types.AutoCertConfig) *Config {
	if cfg.CertPath == "" {
		cfg.CertPath = CertFileDefault
	}
	if cfg.KeyPath == "" {
		cfg.KeyPath = KeyFileDefault
	}
	if cfg.Provider == "" {
		cfg.Provider = ProviderLocal
	}
	return (*Config)(cfg)
}

func (cfg *Config) GetProvider() (provider *Provider, res E.NestedError) {
	b := E.NewBuilder("unable to initialize autocert")
	defer b.To(&res)

	if cfg.Provider != ProviderLocal {
		if len(cfg.Domains) == 0 {
			b.Addf("%s", "no domains specified")
		}
		if cfg.Provider == "" {
			b.Addf("%s", "no provider specified")
		}
		if cfg.Email == "" {
			b.Addf("%s", "no email specified")
		}
		// check if provider is implemented
		_, ok := providersGenMap[cfg.Provider]
		if !ok {
			b.Addf("unknown provider: %q", cfg.Provider)
		}
	}

	if b.HasError() {
		return
	}

	privKey, err := E.Check(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	if err.HasError() {
		b.Add(E.FailWith("generate private key", err))
		return
	}

	user := &User{
		Email: cfg.Email,
		key:   privKey,
	}

	legoCfg := lego.NewConfig(user)
	legoCfg.Certificate.KeyType = certcrypto.RSA2048

	provider = &Provider{
		cfg:     cfg,
		user:    user,
		legoCfg: legoCfg,
	}

	return
}
