package autocert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/sirupsen/logrus"
	E "github.com/yusing/go-proxy/error"
	M "github.com/yusing/go-proxy/models"
	"github.com/yusing/go-proxy/utils"
)

type Provider struct {
	cfg     *Config
	user    *User
	legoCfg *lego.Config
	client  *lego.Client

	tlsCert      *tls.Certificate
	certExpiries CertExpiries
	mutex        sync.Mutex
}

type ProviderGenerator func(M.AutocertProviderOpt) (challenge.Provider, error)
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
	ne := E.Failure("obtain certificate")

	client := p.client
	if p.user.Registration == nil {
		reg, err := E.Check(client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true}))
		if err.IsNotNil() {
			return ne.With(E.Failure("register account").With(err))
		}
		p.user.Registration = reg
	}
	req := certificate.ObtainRequest{
		Domains: p.cfg.Domains,
		Bundle:  true,
	}
	cert, err := E.Check(client.Certificate.Obtain(req))
	if err.IsNotNil() {
		return ne.With(err)
	}
	err = p.saveCert(cert)
	if err.IsNotNil() {
		return ne.With(E.Failure("save certificate").With(err))
	}
	tlsCert, err := E.Check(tls.X509KeyPair(cert.Certificate, cert.PrivateKey))
	if err.IsNotNil() {
		return ne.With(E.Failure("parse obtained certificate").With(err))
	}
	expiries, err := getCertExpiries(&tlsCert)
	if err.IsNotNil() {
		return ne.With(E.Failure("get certificate expiry").With(err))
	}
	p.tlsCert = &tlsCert
	p.certExpiries = expiries
	return E.Nil()
}

func (p *Provider) LoadCert() E.NestedError {
	cert, err := E.Check(tls.LoadX509KeyPair(p.cfg.CertPath, p.cfg.KeyPath))
	if err.IsNotNil() {
		return err
	}
	expiries, err := getCertExpiries(&cert)
	if err.IsNotNil() {
		return err
	}
	p.tlsCert = &cert
	p.certExpiries = expiries
	p.renewIfNeeded()
	return E.Nil()
}

func (p *Provider) ShouldRenewOn() time.Time {
	for _, expiry := range p.certExpiries {
		return expiry.AddDate(0, -1, 0)
	}
	// this line should never be reached
	panic("no certificate available")
}

func (p *Provider) ScheduleRenewal(ctx context.Context) {
	if p.GetName() == ProviderLocal {
		return
	}

	logger.Debug("starting renewal scheduler")
	defer logger.Debug("renewal scheduler stopped")

	stop := make(chan struct{})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			t := time.Until(p.ShouldRenewOn())
			Logger.Infof("next renewal in %v", t.Round(time.Second))
			go func() {
				<-time.After(t)
				close(stop)
			}()
			select {
			case <-ctx.Done():
				return
			case <-stop:
				if err := p.renewIfNeeded(); err.IsNotNil() {
					Logger.Fatal(err)
				}
			}
		}
	}
}

func (p *Provider) saveCert(cert *certificate.Resource) E.NestedError {
	err := os.WriteFile(p.cfg.KeyPath, cert.PrivateKey, 0600) // -rw-------
	if err != nil {
		return E.Failure("write key file").With(err)
	}
	err = os.WriteFile(p.cfg.CertPath, cert.Certificate, 0644) // -rw-r--r--
	if err != nil {
		return E.Failure("write cert file").With(err)
	}
	return E.Nil()
}

func (p *Provider) needRenewal() bool {
	expired := time.Now().After(p.ShouldRenewOn())
	if expired {
		return true
	}
	if len(p.cfg.Domains) != len(p.certExpiries) {
		return true
	}
	wantedDomains := make([]string, len(p.cfg.Domains))
	certDomains := make([]string, len(p.certExpiries))
	copy(wantedDomains, p.cfg.Domains)
	i := 0
	for domain := range p.certExpiries {
		certDomains[i] = domain
		i++
	}
	slices.Sort(wantedDomains)
	slices.Sort(certDomains)
	for i, domain := range certDomains {
		if domain != wantedDomains[i] {
			return true
		}
	}
	return false
}

func (p *Provider) renewIfNeeded() E.NestedError {
	if !p.needRenewal() {
		return E.Nil()
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.needRenewal() {
		return E.Nil()
	}

	trials := 0
	for {
		err := p.ObtainCert()
		if err.IsNotNil() {
			return E.Nil()
		}
		trials++
		if trials > 3 {
			return E.Failure("renew certificate").With(err)
		}
		time.Sleep(5 * time.Second)
	}
}

func getCertExpiries(cert *tls.Certificate) (CertExpiries, E.NestedError) {
	r := make(CertExpiries, len(cert.Certificate))
	for _, cert := range cert.Certificate {
		x509Cert, err := E.Check(x509.ParseCertificate(cert))
		if err.IsNotNil() {
			return nil, E.Failure("parse certificate").With(err)
		}
		if x509Cert.IsCA {
			continue
		}
		r[x509Cert.Subject.CommonName] = x509Cert.NotAfter
	}
	return r, E.Nil()
}

func setOptions[T interface{}](cfg *T, opt M.AutocertProviderOpt) E.NestedError {
	for k, v := range opt {
		err := utils.SetFieldFromSnake(cfg, k, v)
		if err.IsNotNil() {
			return E.Failure("set autocert option").Subject(k).With(err)
		}
	}
	return E.Nil()
}

func providerGenerator[CT any, PT challenge.Provider](
	defaultCfg func() *CT,
	newProvider func(*CT) (PT, error),
) ProviderGenerator {
	return func(opt M.AutocertProviderOpt) (challenge.Provider, error) {
		cfg := defaultCfg()
		err := setOptions(cfg, opt)
		if err.IsNotNil() {
			return nil, err
		}
		p, err := E.Check(newProvider(cfg))
		if err.IsNotNil() {
			return nil, err
		}
		return p, nil
	}
}

var logger = logrus.WithField("?", "autocert")
