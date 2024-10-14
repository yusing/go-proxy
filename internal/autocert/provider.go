package autocert

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path"
	"reflect"
	"sort"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/yusing/go-proxy/internal/common"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/types"
	U "github.com/yusing/go-proxy/internal/utils"
)

type (
	Provider struct {
		cfg     *Config
		user    *User
		legoCfg *lego.Config
		client  *lego.Client

		tlsCert      *tls.Certificate
		certExpiries CertExpiries
	}
	ProviderGenerator func(types.AutocertProviderOpt) (challenge.Provider, E.NestedError)

	CertExpiries map[string]time.Time
)

func (p *Provider) GetCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if p.tlsCert == nil {
		return nil, ErrGetCertFailure
	}
	return p.tlsCert, nil
}

func (p *Provider) GetName() string {
	return p.cfg.Provider
}

func (p *Provider) GetCertPath() string {
	return p.cfg.CertPath
}

func (p *Provider) GetKeyPath() string {
	return p.cfg.KeyPath
}

func (p *Provider) GetExpiries() CertExpiries {
	return p.certExpiries
}

func (p *Provider) ObtainCert() (res E.NestedError) {
	b := E.NewBuilder("failed to obtain certificate")
	defer b.To(&res)

	if p.cfg.Provider == ProviderLocal {
		return nil
	}

	if p.client == nil {
		if err := p.initClient(); err.HasError() {
			b.Add(E.FailWith("init autocert client", err))
			return
		}
	}

	if p.user.Registration == nil {
		if err := p.registerACME(); err.HasError() {
			b.Add(E.FailWith("register ACME", err))
			return
		}
	}

	client := p.client
	req := certificate.ObtainRequest{
		Domains: p.cfg.Domains,
		Bundle:  true,
	}
	cert, err := E.Check(client.Certificate.Obtain(req))
	if err.HasError() {
		b.Add(err)
		return
	}

	if err = p.saveCert(cert); err.HasError() {
		b.Add(E.FailWith("save certificate", err))
		return
	}

	tlsCert, err := E.Check(tls.X509KeyPair(cert.Certificate, cert.PrivateKey))
	if err.HasError() {
		b.Add(E.FailWith("parse obtained certificate", err))
		return
	}

	expiries, err := getCertExpiries(&tlsCert)
	if err.HasError() {
		b.Add(E.FailWith("get certificate expiry", err))
		return
	}
	p.tlsCert = &tlsCert
	p.certExpiries = expiries

	return nil
}

func (p *Provider) LoadCert() E.NestedError {
	cert, err := E.Check(tls.LoadX509KeyPair(p.cfg.CertPath, p.cfg.KeyPath))
	if err.HasError() {
		return err
	}
	expiries, err := getCertExpiries(&cert)
	if err.HasError() {
		return err
	}
	p.tlsCert = &cert
	p.certExpiries = expiries

	logger.Infof("next renewal in %v", U.FormatDuration(time.Until(p.ShouldRenewOn())))
	return p.renewIfNeeded()
}

func (p *Provider) ShouldRenewOn() time.Time {
	for _, expiry := range p.certExpiries {
		return expiry.AddDate(0, -1, 0) // 1 month before
	}
	// this line should never be reached
	panic("no certificate available")
}

func (p *Provider) ScheduleRenewal() {
	if p.GetName() == ProviderLocal {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	task := common.NewTask("cert renew scheduler")
	defer task.Finished()

	for {
		select {
		case <-task.Context().Done():
			return
		case <-ticker.C: // check every 5 seconds
			if err := p.renewIfNeeded(); err.HasError() {
				logger.Warn(err)
			}
		}
	}
}

func (p *Provider) initClient() E.NestedError {
	legoClient, err := E.Check(lego.NewClient(p.legoCfg))
	if err.HasError() {
		return E.FailWith("create lego client", err)
	}

	legoProvider, err := providersGenMap[p.cfg.Provider](p.cfg.Options)
	if err.HasError() {
		return E.FailWith("create lego provider", err)
	}

	err = E.From(legoClient.Challenge.SetDNS01Provider(legoProvider))
	if err.HasError() {
		return E.FailWith("set challenge provider", err)
	}

	p.client = legoClient
	return nil
}

func (p *Provider) registerACME() E.NestedError {
	if p.user.Registration != nil {
		return nil
	}
	reg, err := E.Check(p.client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true}))
	if err.HasError() {
		return err
	}
	p.user.Registration = reg

	return nil
}

func (p *Provider) saveCert(cert *certificate.Resource) E.NestedError {
	/* This should have been done in setup
	but double check is always a good choice.*/
	_, err := os.Stat(path.Dir(p.cfg.CertPath))
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path.Dir(p.cfg.CertPath), 0o755); err != nil {
				return E.FailWith("create cert directory", err)
			}
		} else {
			return E.FailWith("stat cert directory", err)
		}
	}
	err = os.WriteFile(p.cfg.KeyPath, cert.PrivateKey, 0o600) // -rw-------
	if err != nil {
		return E.FailWith("write key file", err)
	}
	err = os.WriteFile(p.cfg.CertPath, cert.Certificate, 0o644) // -rw-r--r--
	if err != nil {
		return E.FailWith("write cert file", err)
	}
	return nil
}

func (p *Provider) certState() CertState {
	if time.Now().After(p.ShouldRenewOn()) {
		return CertStateExpired
	}

	certDomains := make([]string, len(p.certExpiries))
	wantedDomains := make([]string, len(p.cfg.Domains))
	i := 0
	for domain := range p.certExpiries {
		certDomains[i] = domain
		i++
	}
	copy(wantedDomains, p.cfg.Domains)
	sort.Strings(wantedDomains)
	sort.Strings(certDomains)

	if !reflect.DeepEqual(certDomains, wantedDomains) {
		logger.Infof("cert domains mismatch: %v != %v", certDomains, p.cfg.Domains)
		return CertStateMismatch
	}

	return CertStateValid
}

func (p *Provider) renewIfNeeded() E.NestedError {
	if p.cfg.Provider == ProviderLocal {
		return nil
	}

	switch p.certState() {
	case CertStateExpired:
		logger.Info("certs expired, renewing")
	case CertStateMismatch:
		logger.Info("cert domains mismatch with config, renewing")
	default:
		return nil
	}

	if err := p.ObtainCert(); err.HasError() {
		return E.FailWith("renew certificate", err)
	}
	return nil
}

func getCertExpiries(cert *tls.Certificate) (CertExpiries, E.NestedError) {
	r := make(CertExpiries, len(cert.Certificate))
	for _, cert := range cert.Certificate {
		x509Cert, err := E.Check(x509.ParseCertificate(cert))
		if err.HasError() {
			return nil, E.FailWith("parse certificate", err)
		}
		if x509Cert.IsCA {
			continue
		}
		r[x509Cert.Subject.CommonName] = x509Cert.NotAfter
		for i := range x509Cert.DNSNames {
			r[x509Cert.DNSNames[i]] = x509Cert.NotAfter
		}
	}
	return r, nil
}

func providerGenerator[CT any, PT challenge.Provider](
	defaultCfg func() *CT,
	newProvider func(*CT) (PT, error),
) ProviderGenerator {
	return func(opt types.AutocertProviderOpt) (challenge.Provider, E.NestedError) {
		cfg := defaultCfg()
		err := U.Deserialize(opt, cfg)
		if err.HasError() {
			return nil, err
		}
		p, err := E.Check(newProvider(cfg))
		if err.HasError() {
			return nil, err
		}
		return p, nil
	}
}
