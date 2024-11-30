package autocert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"os"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
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
	if cfg.ACMEKeyPath == "" {
		cfg.ACMEKeyPath = ACMEKeyFileDefault
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

	var privKey *ecdsa.PrivateKey
	var err error

	if cfg.Provider != ProviderLocal {
		if privKey, err = cfg.loadACMEKey(); err != nil {
			logging.Info().Err(err).Msg("load ACME private key failed")
			logging.Info().Msg("generate new ACME private key")
			privKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return nil, E.New("generate ACME private key").With(err)
			}
			if err = cfg.saveACMEKey(privKey); err != nil {
				return nil, E.New("save ACME private key").With(err)
			}
		}
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

func (cfg *Config) loadACMEKey() (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(cfg.ACMEKeyPath)
	if err != nil {
		return nil, err
	}
	return x509.ParseECPrivateKey(data)
}

func (cfg *Config) saveACMEKey(key *ecdsa.PrivateKey) error {
	data, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.ACMEKeyPath, data, 0o600)
}
