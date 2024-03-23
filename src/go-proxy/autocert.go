package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/registration"
)

type AutoCertConfig struct {
	Email    string
	Domains  []string `yaml:",flow"`
	Provider string
	Options  map[string]string `yaml:",flow"`
}

type AutoCertUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *AutoCertUser) GetEmail() string {
	return u.Email
}
func (u *AutoCertUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *AutoCertUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

type AutoCertProvider interface {
	GetCert(*tls.ClientHelloInfo) (*tls.Certificate, error)
	GetName() string
	GetExpiry() time.Time
	LoadCert() bool
	ObtainCert() error

	needRenew() bool
}

func (cfg AutoCertConfig) GetProvider() (AutoCertProvider, error) {
	if len(cfg.Domains) == 0 {
		return nil, fmt.Errorf("no domains specified")
	}
	if cfg.Provider == "" {
		return nil, fmt.Errorf("no provider specified")
	}
	if cfg.Email == "" {
		return nil, fmt.Errorf("no email specified")
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("unable to generate private key: %v", err)
	}
	user := &AutoCertUser{
		Email: cfg.Email,
		key:   privKey,
	}
	legoCfg := lego.NewConfig(user)
	legoCfg.Certificate.KeyType = certcrypto.RSA2048
	legoClient, err := lego.NewClient(legoCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create lego client: %v", err)
	}
	base := &AutoCertProviderBase{
		name:    cfg.Provider,
		cfg:     cfg,
		user:    user,
		legoCfg: legoCfg,
		client:  legoClient,
	}
	switch cfg.Provider {
	case "cloudflare":
		return NewAutoCertCFProvider(base, cfg.Options)
	}
	return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
}

type AutoCertProviderBase struct {
	name    string
	cfg     AutoCertConfig
	user    *AutoCertUser
	legoCfg *lego.Config
	client  *lego.Client

	tlsCert *tls.Certificate
	expiry  time.Time
	mutex   sync.Mutex
}

func (p *AutoCertProviderBase) GetCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if p.tlsCert == nil {
		aclog.Fatal("no certificate available")
	}
	if p.needRenew() {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		if p.needRenew() {
			err := p.ObtainCert()
			if err != nil {
				return nil, err
			}
		}
	}
	return p.tlsCert, nil
}

func (p *AutoCertProviderBase) GetName() string {
	return p.name
}

func (p *AutoCertProviderBase) GetExpiry() time.Time {
	return p.expiry
}

func (p *AutoCertProviderBase) ObtainCert() error {
	client := p.client
	if p.user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return err
		}
		p.user.Registration = reg
	}
	req := certificate.ObtainRequest{
		Domains: p.cfg.Domains,
		Bundle:  true,
	}
	cert, err := client.Certificate.Obtain(req)
	if err != nil {
		return err
	}
	err = p.saveCert(cert)
	if err != nil {
		return err
	}
	tlsCert, err := tls.X509KeyPair(cert.Certificate, cert.PrivateKey)
	if err != nil {
		return err
	}
	p.tlsCert = &tlsCert
	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[len(tlsCert.Certificate)-1])
	if err != nil {
		return err
	}
	p.expiry = x509Cert.NotAfter
	return nil
}

func (p *AutoCertProviderBase) LoadCert() bool {
	cert, err := tls.LoadX509KeyPair(certFileDefault, keyFileDefault)
	if err != nil {
		return false
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[len(cert.Certificate)-1])
	if err != nil {
		return false
	}
	p.tlsCert = &cert
	p.expiry = x509Cert.NotAfter
	return true
}

func (p *AutoCertProviderBase) saveCert(cert *certificate.Resource) error {
	err := os.MkdirAll(path.Dir(certFileDefault), 0644)
	if err != nil {
		return fmt.Errorf("unable to create cert directory: %v", err)
	}
	err = os.WriteFile(keyFileDefault, cert.PrivateKey, 0600) // -rw-------
	if err != nil {
		return fmt.Errorf("unable to write key file: %v", err)
	}
	err = os.WriteFile(certFileDefault, cert.Certificate, 0644) // -rw-r--r--
	if err != nil {
		return fmt.Errorf("unable to write cert file: %v", err)
	}
	return nil
}

func (p *AutoCertProviderBase) needRenew() bool {
	return p.expiry.Before(time.Now().Add(24 * time.Hour))
}

type AutoCertCFProvider struct {
	*AutoCertProviderBase
	*cloudflare.Config
}

func NewAutoCertCFProvider(base *AutoCertProviderBase, opt map[string]string) (*AutoCertCFProvider, error) {
	p := &AutoCertCFProvider{
		base,
		cloudflare.NewDefaultConfig(),
	}
	err := setOptions(p.Config, opt)
	if err != nil {
		return nil, err
	}
	legoProvider, err := cloudflare.NewDNSProviderConfig(p.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to create cloudflare provider: %v", err)
	}
	err = p.client.Challenge.SetDNS01Provider(legoProvider)
	if err != nil {
		return nil, fmt.Errorf("unable to set challenge provider: %v", err)
	}
	return p, nil
}

func setOptions[T interface{}](cfg *T, opt map[string]string) error {
	for k, v := range opt {
		err := SetFieldFromSnake(cfg, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
