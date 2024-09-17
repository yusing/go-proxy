package autocert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"reflect"
	"sort"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	U "github.com/yusing/go-proxy/utils"
)

type Provider struct {
	cfg     *Config
	user    *User
	legoCfg *lego.Config
	client  *lego.Client

	tlsCert      *tls.Certificate
	certExpiries CertExpiries
}

type ProviderGenerator func(M.AutocertProviderOpt) (challenge.Provider, E.NestedError)
type CertExpiries map[string]time.Time

func (p *Provider) GetCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if p.tlsCert == nil {
		return nil, E.Failure("get certificate")
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

func (p *Provider) ObtainCert() E.NestedError {
	if p.cfg.Provider == ProviderLocal {
		return E.FailureWhy("obtain cert", "provider is set to \"local\"")
	}

	if p.client == nil {
		if err := p.initClient(); err.HasError() {
			return E.Failure("obtain cert").With(err)
		}
	}

	ne := E.Failure("obtain certificate")

	client := p.client
	if p.user.Registration == nil {
		if err := p.loadRegistration(); err.HasError() {
			ne = ne.With(err)
			if err := p.registerACME(); err.HasError() {
				return ne.With(err)
			}
		}
	}
	req := certificate.ObtainRequest{
		Domains: p.cfg.Domains,
		Bundle:  true,
	}
	cert, err := E.Check(client.Certificate.Obtain(req))
	if err.HasError() {
		return ne.With(err)
	}
	err = p.saveCert(cert)
	if err.HasError() {
		return ne.With(E.Failure("save certificate").With(err))
	}
	tlsCert, err := E.Check(tls.X509KeyPair(cert.Certificate, cert.PrivateKey))
	if err.HasError() {
		return ne.With(E.Failure("parse obtained certificate").With(err))
	}
	expiries, err := getCertExpiries(&tlsCert)
	if err.HasError() {
		return ne.With(E.Failure("get certificate expiry").With(err))
	}
	p.tlsCert = &tlsCert
	p.certExpiries = expiries

	return E.Nil()
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

func (p *Provider) ScheduleRenewal(ctx context.Context) {
	if p.GetName() == ProviderLocal {
		return
	}

	logger.Debug("started renewal scheduler")
	defer logger.Debug("renewal scheduler stopped")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
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
		return E.Failure("create lego client").With(err)
	}

	legoProvider, err := providersGenMap[p.cfg.Provider](p.cfg.Options)
	if err.HasError() {
		return E.Failure("create lego provider").With(err)
	}

	err = E.From(legoClient.Challenge.SetDNS01Provider(legoProvider))
	if err.HasError() {
		return E.Failure("set challenge provider").With(err)
	}

	p.client = legoClient
	return E.Nil()
}

func (p *Provider) registerACME() E.NestedError {
	if p.user.Registration != nil {
		return E.Nil()
	}
	reg, err := E.Check(p.client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true}))
	if err.HasError() {
		return E.Failure("register ACME").With(err)
	}
	p.user.Registration = reg

	if err := p.saveRegistration(); err.HasError() {
		logger.Warn(err)
	}
	return E.Nil()
}

func (p *Provider) loadRegistration() E.NestedError {
	if p.user.Registration != nil {
		return E.Nil()
	}
	reg := &registration.Resource{}
	err := U.LoadJson(RegistrationFile, reg)
	if err.HasError() {
		return E.Failure("parse registration file").With(err)
	}
	p.user.Registration = reg
	return E.Nil()
}

func (p *Provider) saveRegistration() E.NestedError {
	return U.SaveJson(RegistrationFile, p.user.Registration, 0o600)
}

func (p *Provider) saveCert(cert *certificate.Resource) E.NestedError {
	err := os.WriteFile(p.cfg.KeyPath, cert.PrivateKey, 0o600) // -rw-------
	if err != nil {
		return E.Failure("write key file").With(err)
	}
	err = os.WriteFile(p.cfg.CertPath, cert.Certificate, 0o644) // -rw-r--r--
	if err != nil {
		return E.Failure("write cert file").With(err)
	}
	return E.Nil()
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
		logger.Debugf("cert domains mismatch: %v != %v", certDomains, p.cfg.Domains)
		return CertStateMismatch
	}

	return CertStateValid
}

func (p *Provider) renewIfNeeded() E.NestedError {
	switch p.certState() {
	case CertStateExpired:
		logger.Info("certs expired, renewing")
	case CertStateMismatch:
		logger.Info("cert domains mismatch with config, renewing")
	default:
		return E.Nil()
	}

	if err := p.ObtainCert(); err.HasError() {
		return E.Failure("renew certificate").With(err)
	}
	return E.Nil()
}

func getCertExpiries(cert *tls.Certificate) (CertExpiries, E.NestedError) {
	r := make(CertExpiries, len(cert.Certificate))
	for _, cert := range cert.Certificate {
		x509Cert, err := E.Check(x509.ParseCertificate(cert))
		if err.HasError() {
			return nil, E.Failure("parse certificate").With(err)
		}
		if x509Cert.IsCA {
			continue
		}
		r[x509Cert.Subject.CommonName] = x509Cert.NotAfter
		for i := range x509Cert.DNSNames {
			r[x509Cert.DNSNames[i]] = x509Cert.NotAfter
		}
	}
	return r, E.Nil()
}

func providerGenerator[CT any, PT challenge.Provider](
	defaultCfg func() *CT,
	newProvider func(*CT) (PT, error),
) ProviderGenerator {
	return func(opt M.AutocertProviderOpt) (challenge.Provider, E.NestedError) {
		cfg := defaultCfg()
		err := U.Deserialize(opt, cfg)
		if err.HasError() {
			return nil, err
		}
		p, err := E.Check(newProvider(cfg))
		if err.HasError() {
			return nil, err
		}
		return p, E.Nil()
	}
}
