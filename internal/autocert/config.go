package autocert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/utils/strutils"

	"github.com/yusing/go-proxy/internal/config/types"
)

type Config types.AutoCertConfig

var (
	ErrMissingDomain   = E.New("missing field 'domains'")
	ErrMissingEmail    = E.New("missing field 'email'")
	ErrMissingProvider = E.New("missing field 'provider'")
	ErrUnknownProvider = E.New("unknown provider")
)

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

func (cfg *Config) GetProvider() (*Provider, E.Error) {
	b := E.NewBuilder("autocert errors")

	if cfg.Provider != ProviderLocal {
		if len(cfg.Domains) == 0 {
			b.Add(ErrMissingDomain)
		}
		if cfg.Provider == "" {
			b.Add(ErrMissingProvider)
		}
		if cfg.Email == "" {
			b.Add(ErrMissingEmail)
		}
		// check if provider is implemented
		_, ok := providersGenMap[cfg.Provider]
		if !ok {
			b.Add(ErrUnknownProvider.
				Subject(cfg.Provider).
				Withf(strutils.DoYouMean(utils.NearestField(cfg.Provider, providersGenMap))))
		}
	}

	if b.HasError() {
		return nil, b.Error()
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		b.Addf("generate private key: %w", err)
		return nil, b.Error()
	}

	user := &User{
		Email: cfg.Email,
		key:   privKey,
	}

	legoCfg := lego.NewConfig(user)
	legoCfg.Certificate.KeyType = certcrypto.RSA2048

	return &Provider{
		cfg:     cfg,
		user:    user,
		legoCfg: legoCfg,
	}, nil
}
