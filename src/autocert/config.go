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
	}

	gen, ok := providersGenMap[cfg.Provider]
	if !ok {
		errors.Addf("unknown provider: %q", cfg.Provider)
	}
	if err := errors.Build(); err.IsNotNil() {
		return nil, err
	}

	privKey, err := E.Check(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	if err.IsNotNil() {
		return nil, E.Failure("generate private key").With(err)
	}
	user := &User{
		Email: cfg.Email,
		key:   privKey,
	}
	legoCfg := lego.NewConfig(user)
	legoCfg.Certificate.KeyType = certcrypto.RSA2048
	legoClient, err := E.Check(lego.NewClient(legoCfg))
	if err.IsNotNil() {
		return nil, E.Failure("create lego client").With(err)
	}
	base := &Provider{
		cfg:     cfg,
		user:    user,
		legoCfg: legoCfg,
		client:  legoClient,
	}
	legoProvider, err := E.Check(gen(cfg.Options))
	if err.IsNotNil() {
		return nil, E.Failure("create lego provider").With(err)
	}
	err = E.From(legoClient.Challenge.SetDNS01Provider(legoProvider))
	if err.IsNotNil() {
		return nil, E.Failure("set challenge provider").With(err)
	}
	return base, E.Nil()
}
