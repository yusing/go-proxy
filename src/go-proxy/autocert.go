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
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/registration"
)

type ProviderOptions = map[string]string
type ProviderGenerator = func(ProviderOptions) (challenge.Provider, error)
type CertExpiries = map[string]time.Time

type AutoCertConfig struct {
	Email    string
	Domains  []string `yaml:",flow"`
	Provider string
	Options  ProviderOptions `yaml:",flow"`
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
	ObtainCert() error
	RenewalOn() time.Time
	ScheduleRenewal()
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
	base := &autoCertProvider{
		name:    cfg.Provider,
		cfg:     cfg,
		user:    user,
		legoCfg: legoCfg,
		client:  legoClient,
	}
	gen, ok := providersGenMap[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
	legoProvider, err := gen(cfg.Options)
	if err != nil {
		return nil, fmt.Errorf("unable to create provider: %v", err)
	}
	err = legoClient.Challenge.SetDNS01Provider(legoProvider)
	if err != nil {
		return nil, fmt.Errorf("unable to set challenge provider: %v", err)
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
		aclog.Fatal("no certificate available")
	}
	return p.tlsCert, nil
}

func (p *autoCertProvider) GetName() string {
	return p.name
}

func (p *autoCertProvider) GetExpiries() CertExpiries {
	return p.certExpiries
}

func (p *autoCertProvider) ObtainCert() error {
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
	expiries, err := getCertExpiries(&tlsCert)
	if err != nil {
		return err
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

func (p *autoCertProvider) saveCert(cert *certificate.Resource) error {
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

func (p *autoCertProvider) needRenewal() bool {
	return time.Now().After(p.RenewalOn())
}

func (p *autoCertProvider) renewIfNeeded() error {
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
			return fmt.Errorf("unable to renew certificate: %v after 3 trials", err)
		}
		aclog.Errorf("failed to renew certificate: %v, trying again in 5 seconds", err)
		time.Sleep(5 * time.Second)
	}
}

func providerGenerator[CT interface{}, PT challenge.Provider](defaultCfg func() *CT, newProvider func(*CT) (PT, error)) ProviderGenerator {
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
			return nil, err
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
		err := SetFieldFromSnake(cfg, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

var providersGenMap = map[string]ProviderGenerator{
	"cloudflare": providerGenerator(cloudflare.NewDefaultConfig, cloudflare.NewDNSProviderConfig),
}
