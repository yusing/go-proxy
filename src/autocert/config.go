package autocert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
)

type Config M.AutoCertConfig

func NewConfig(cfg *M.AutoCertConfig) *Config {
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

func (cfg *Config) GetProvider() (*Provider, E.NestedError) {
	errors := E.NewBuilder("cannot create autocert provider")

	if cfg.Provider != ProviderLocal {
		if len(cfg.Domains) == 0 {
			errors.Addf("no domains specified")
		}
		if cfg.Provider == "" {
			errors.Addf("no provider specified")
		}
		if cfg.Email == "" {
			errors.Addf("no email specified")
		}
		// check if provider is implemented
		_, ok := providersGenMap[cfg.Provider]
		if !ok {
			errors.Addf("unknown provider: %q", cfg.Provider)
		}
	}

	if err := errors.Build(); err.HasError() {
		return nil, err
	}

	privKey, err := E.Check(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	if err.HasError() {
		return nil, E.Failure("generate private key").With(err)
	}

	user := &User{
		Email: cfg.Email,
		key:   privKey,
	}

	legoCfg := lego.NewConfig(user)
	legoCfg.Certificate.KeyType = certcrypto.RSA2048

	base := &Provider{
		cfg:     cfg,
		user:    user,
		legoCfg: legoCfg,
	}

	return base, E.Nil()
}
