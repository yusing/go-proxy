package autocert

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"path"
	"reflect"
	"runtime"
	"sort"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/yusing/go-proxy/internal/config/types"
	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/task"
	U "github.com/yusing/go-proxy/internal/utils"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	Provider struct {
		cfg     *Config
		user    *User
		legoCfg *lego.Config
		client  *lego.Client

		legoCert     *certificate.Resource
		tlsCert      *tls.Certificate
		certExpiries CertExpiries
	}
	ProviderGenerator func(types.AutocertProviderOpt) (challenge.Provider, E.Error)

	CertExpiries map[string]time.Time
)

var ErrGetCertFailure = errors.New("get certificate failed")

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

func (p *Provider) ObtainCert() E.Error {
	if p.cfg.Provider == ProviderLocal {
		return nil
	}

	if p.client == nil {
		if err := p.initClient(); err != nil {
			return err
		}
	}

	if p.user.Registration == nil {
		if err := p.registerACME(); err != nil {
			return E.From(err)
		}
	}

	var cert *certificate.Resource
	var err error

	if p.legoCert != nil {
		cert, err = p.client.Certificate.RenewWithOptions(*p.legoCert, &certificate.RenewOptions{
			Bundle: true,
		})
		if err != nil {
			p.legoCert = nil
			logger.Err(err).Msg("cert renew failed, fallback to obtain")
		} else {
			p.legoCert = cert
		}
	}

	if cert == nil {
		cert, err = p.client.Certificate.Obtain(certificate.ObtainRequest{
			Domains: p.cfg.Domains,
			Bundle:  true,
		})
		if err != nil {
			return E.From(err)
		}
	}

	if err = p.saveCert(cert); err != nil {
		return E.From(err)
	}

	tlsCert, err := tls.X509KeyPair(cert.Certificate, cert.PrivateKey)
	if err != nil {
		return E.From(err)
	}

	expiries, err := getCertExpiries(&tlsCert)
	if err != nil {
		return E.From(err)
	}
	p.tlsCert = &tlsCert
	p.certExpiries = expiries

	return nil
}

func (p *Provider) LoadCert() E.Error {
	cert, err := tls.LoadX509KeyPair(p.cfg.CertPath, p.cfg.KeyPath)
	if err != nil {
		return E.Errorf("load SSL certificate: %w", err)
	}
	expiries, err := getCertExpiries(&cert)
	if err != nil {
		return E.Errorf("parse SSL certificate: %w", err)
	}
	p.tlsCert = &cert
	p.certExpiries = expiries

	logger.Info().Msgf("next renewal in %v", strutils.FormatDuration(time.Until(p.ShouldRenewOn())))
	return p.renewIfNeeded()
}

// ShouldRenewOn returns the time at which the certificate should be renewed.
func (p *Provider) ShouldRenewOn() time.Time {
	for _, expiry := range p.certExpiries {
		return expiry.AddDate(0, -1, 0) // 1 month before
	}
	// this line should never be reached
	panic("no certificate available")
}

func (p *Provider) ScheduleRenewal(parent task.Parent) {
	if p.GetName() == ProviderLocal {
		return
	}
	go func() {
		lastErrOn := time.Time{}
		renewalTime := p.ShouldRenewOn()
		timer := time.NewTimer(time.Until(renewalTime))
		defer timer.Stop()

		task := parent.Subtask("cert-renew-scheduler")
		defer task.Finish(nil)

		for {
			select {
			case <-task.Context().Done():
				return
			case <-timer.C:
				// Retry after 1 hour on failure
				if time.Now().Before(lastErrOn.Add(time.Hour)) {
					continue
				}
				if err := p.renewIfNeeded(); err != nil {
					E.LogWarn("cert renew failed", err, &logger)
					lastErrOn = time.Now()
					continue
				}
				// Reset on success
				lastErrOn = time.Time{}
				renewalTime = p.ShouldRenewOn()
				timer.Reset(time.Until(renewalTime))
			default:
				// Allow other tasks to run
				runtime.Gosched()
			}
		}
	}()
}

func (p *Provider) initClient() E.Error {
	legoClient, err := lego.NewClient(p.legoCfg)
	if err != nil {
		return E.From(err)
	}

	generator := providersGenMap[p.cfg.Provider]
	legoProvider, pErr := generator(p.cfg.Options)
	if pErr != nil {
		return pErr
	}

	err = legoClient.Challenge.SetDNS01Provider(legoProvider)
	if err != nil {
		return E.From(err)
	}

	p.client = legoClient
	return nil
}

func (p *Provider) registerACME() error {
	if p.user.Registration != nil {
		return nil
	}
	if reg, err := p.client.Registration.ResolveAccountByKey(); err == nil {
		p.user.Registration = reg
		logger.Info().Msg("reused acme registration from private key")
		return nil
	}

	reg, err := p.client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return err
	}
	p.user.Registration = reg
	logger.Info().Interface("reg", reg).Msg("acme registered")
	return nil
}

func (p *Provider) saveCert(cert *certificate.Resource) error {
	/* This should have been done in setup
	but double check is always a good choice.*/
	_, err := os.Stat(path.Dir(p.cfg.CertPath))
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path.Dir(p.cfg.CertPath), 0o755); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	err = os.WriteFile(p.cfg.KeyPath, cert.PrivateKey, 0o600) // -rw-------
	if err != nil {
		return err
	}

	err = os.WriteFile(p.cfg.CertPath, cert.Certificate, 0o644) // -rw-r--r--
	if err != nil {
		return err
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
		logger.Info().Msgf("cert domains mismatch: %v != %v", certDomains, p.cfg.Domains)
		return CertStateMismatch
	}

	return CertStateValid
}

func (p *Provider) renewIfNeeded() E.Error {
	if p.cfg.Provider == ProviderLocal {
		return nil
	}

	switch p.certState() {
	case CertStateExpired:
		logger.Info().Msg("certs expired, renewing")
	case CertStateMismatch:
		logger.Info().Msg("cert domains mismatch with config, renewing")
	default:
		return nil
	}

	return p.ObtainCert()
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
	return func(opt types.AutocertProviderOpt) (challenge.Provider, E.Error) {
		cfg := defaultCfg()
		err := U.Deserialize(opt, cfg)
		if err != nil {
			return nil, err
		}
		p, pErr := newProvider(cfg)
		return p, E.From(pErr)
	}
}
