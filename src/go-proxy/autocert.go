package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"os"
	"path"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/clouddns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/registration"
)

type ProviderOptions map[string]string
type ProviderGenerator func(ProviderOptions) (challenge.Provider, error)
type CertExpiries map[string]time.Time

type AutoCertConfig struct {
	Email    string          `json:"email"`
	Domains  []string        `yaml:",flow" json:"domains"`
	Provider string          `json:"provider"`
	Options  ProviderOptions `yaml:",flow" json:"options"`
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
	GetExpiries() CertExpiries
	LoadCert() bool
	ObtainCert() NestedErrorLike
	RenewalOn() time.Time
	ScheduleRenewal()
}

func (cfg AutoCertConfig) GetProvider() (AutoCertProvider, error) {
	ne := NewNestedError("invalid autocert config")

	if len(cfg.Domains) == 0 {
		ne.Extra("no domains specified")
	}
	if cfg.Provider == "" {
		ne.Extra("no provider specified")
	}
	if cfg.Email == "" {
		ne.Extra("no email specified")
	}
	gen, ok := providersGenMap[cfg.Provider]
	if !ok {
		ne.Extraf("unknown provider: %s", cfg.Provider)
	}
	if ne.HasExtras() {
		return nil, ne
	}

	ne = NewNestedError("unable to create provider")
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, ne.With(NewNestedError("unable to generate private key").With(err))
	}
	user := &AutoCertUser{
		Email: cfg.Email,
		key:   privKey,
	}
	legoCfg := lego.NewConfig(user)
	legoCfg.Certificate.KeyType = certcrypto.RSA2048
	legoClient, err := lego.NewClient(legoCfg)
	if err != nil {
		return nil, ne.With(NewNestedError("unable to create lego client").With(err))
	}
	base := &autoCertProvider{
		name:    cfg.Provider,
		cfg:     cfg,
		user:    user,
		legoCfg: legoCfg,
		client:  legoClient,
	}
	legoProvider, err := gen(cfg.Options)
	if err != nil {
		return nil, ne.With(err)
	}
	err = legoClient.Challenge.SetDNS01Provider(legoProvider)
	if err != nil {
		return nil, ne.With(NewNestedError("unable to set challenge provider").With(err))
	}
	return base, nil
}

type autoCertProvider struct {
	name    string
	cfg     AutoCertConfig
	user    *AutoCertUser
	legoCfg *lego.Config
	client  *lego.Client

	tlsCert      *tls.Certificate
	certExpiries CertExpiries
	mutex        sync.Mutex
}

func (p *autoCertProvider) GetCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if p.tlsCert == nil {
		return nil, NewNestedError("no certificate available")
	}
	return p.tlsCert, nil
}

func (p *autoCertProvider) GetName() string {
	return p.name
}

func (p *autoCertProvider) GetExpiries() CertExpiries {
	return p.certExpiries
}

func (p *autoCertProvider) ObtainCert() NestedErrorLike {
	ne := NewNestedError("failed to obtain certificate")

	client := p.client
	if p.user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return ne.With(NewNestedError("failed to register account").With(err))
		}
		p.user.Registration = reg
	}
	req := certificate.ObtainRequest{
		Domains: p.cfg.Domains,
		Bundle:  true,
	}
	cert, err := client.Certificate.Obtain(req)
	if err != nil {
		return ne.With(err)
	}
	err = p.saveCert(cert)
	if err != nil {
		return ne.With(NewNestedError("failed to save certificate").With(err))
	}
	tlsCert, err := tls.X509KeyPair(cert.Certificate, cert.PrivateKey)
	if err != nil {
		return ne.With(NewNestedError("failed to parse obtained certificate").With(err))
	}
	expiries, err := getCertExpiries(&tlsCert)
	if err != nil {
		return ne.With(NewNestedError("failed to get certificate expiry").With(err))
	}
	p.tlsCert = &tlsCert
	p.certExpiries = expiries
	return nil
}

func (p *autoCertProvider) LoadCert() bool {
	cert, err := tls.LoadX509KeyPair(certFileDefault, keyFileDefault)
	if err != nil {
		return false
	}
	expiries, err := getCertExpiries(&cert)
	if err != nil {
		return false
	}
	p.tlsCert = &cert
	p.certExpiries = expiries
	p.renewIfNeeded()
	return true
}

func (p *autoCertProvider) RenewalOn() time.Time {
	t := time.Now().AddDate(0, 0, 3)
	for _, expiry := range p.certExpiries {
		if expiry.Before(t) {
			return time.Now()
		}
		return t
	}
	// this line should never be reached
	panic("no certificate available")
}

func (p *autoCertProvider) ScheduleRenewal() {
	for {
		t := time.Until(p.RenewalOn())
		aclog.Infof("next renewal in %v", t)
		time.Sleep(t)
		err := p.renewIfNeeded()
		if err != nil {
			aclog.Fatal(err)
		}
	}
}

func (p *autoCertProvider) saveCert(cert *certificate.Resource) NestedErrorLike {
	err := os.MkdirAll(path.Dir(certFileDefault), 0644)
	if err != nil {
		return NewNestedError("unable to create cert directory").With(err)
	}
	err = os.WriteFile(keyFileDefault, cert.PrivateKey, 0600) // -rw-------
	if err != nil {
		return NewNestedError("unable to write key file").With(err)
	}
	err = os.WriteFile(certFileDefault, cert.Certificate, 0644) // -rw-r--r--
	if err != nil {
		return NewNestedError("unable to write cert file").With(err)
	}
	return nil
}

func (p *autoCertProvider) needRenewal() bool {
	return time.Now().After(p.RenewalOn())
}

func (p *autoCertProvider) renewIfNeeded() NestedErrorLike {
	if !p.needRenewal() {
		return nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.needRenewal() {
		return nil
	}

	trials := 0
	for {
		err := p.ObtainCert()
		if err == nil {
			return nil
		}
		trials++
		if trials > 3 {
			return NewNestedError("failed to renew certificate after 3 trials").With(err)
		}
		aclog.Errorf("failed to renew certificate: %v, trying again in 5 seconds", err)
		time.Sleep(5 * time.Second)
	}
}

func providerGenerator[CT any, PT challenge.Provider](
	defaultCfg func() *CT,
	newProvider func(*CT) (PT, error),
) ProviderGenerator {
	return func(opt ProviderOptions) (challenge.Provider, error) {
		cfg := defaultCfg()
		err := setOptions(cfg, opt)
		if err != nil {
			return nil, err
		}
		p, err := newProvider(cfg)
		if err != nil {
			return nil, err
		}
		return p, nil
	}
}

func getCertExpiries(cert *tls.Certificate) (CertExpiries, error) {
	r := make(CertExpiries, len(cert.Certificate))
	for _, cert := range cert.Certificate {
		x509Cert, err := x509.ParseCertificate(cert)
		if err != nil {
			return nil, NewNestedError("unable to parse certificate").With(err)
		}
		if x509Cert.IsCA {
			continue
		}
		r[x509Cert.Subject.CommonName] = x509Cert.NotAfter
	}
	return r, nil
}

func setOptions[T interface{}](cfg *T, opt ProviderOptions) error {
	for k, v := range opt {
		err := setFieldFromSnake(cfg, k, v)
		if err != nil {
			return NewNestedError("unable to set option").Subject(k).With(err)
		}
	}
	return nil
}

var providersGenMap = map[string]ProviderGenerator{
	"cloudflare": providerGenerator(cloudflare.NewDefaultConfig, cloudflare.NewDNSProviderConfig),
	"clouddns": providerGenerator(clouddns.NewDefaultConfig, clouddns.NewDNSProviderConfig),
}
