package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/utils"
)

const (
	CertsDNSName = "godoxy.agent"

	caCertPath  = "certs/ca.crt"
	caKeyPath   = "certs/ca.key"
	srvCertPath = "certs/agent.crt"
	srvKeyPath  = "certs/agent.key"
)

func loadCerts(certPath, keyPath string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	return &cert, err
}

func write(b []byte, f *os.File) error {
	_, err := f.Write(b)
	return err
}

func saveCerts(certDER []byte, key *rsa.PrivateKey, certPath, keyPath string) ([]byte, []byte, error) {
	certPEM, keyPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	if certPath == "" || keyPath == "" {
		return certPEM, keyPEM, nil
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		return nil, nil, err
	}
	defer certFile.Close()

	keyFile, err := os.Create(keyPath)
	if err != nil {
		return nil, nil, err
	}
	defer keyFile.Close()

	return certPEM, keyPEM, errors.Join(
		write(certPEM, certFile),
		write(keyPEM, keyFile),
	)
}

func checkExists(certPath, keyPath string) bool {
	certExists, err := utils.FileExists(certPath)
	if err != nil {
		E.LogFatal("cert error", err)
	}
	keyExists, err := utils.FileExists(keyPath)
	if err != nil {
		E.LogFatal("key error", err)
	}
	return certExists && keyExists
}

func InitCerts() (ca *tls.Certificate, srv *tls.Certificate, isNew bool, err error) {
	if checkExists(caCertPath, caKeyPath) && checkExists(srvCertPath, srvKeyPath) {
		logging.Info().Msg("Loading existing certs...")
		ca, err = loadCerts(caCertPath, caKeyPath)
		if err != nil {
			return nil, nil, false, err
		}
		srv, err = loadCerts(srvCertPath, srvKeyPath)
		if err != nil {
			return nil, nil, false, err
		}

		logging.Info().Msg("Verifying agent cert...")

		roots := x509.NewCertPool()
		roots.AddCert(ca.Leaf)

		srvCert, err := x509.ParseCertificate(srv.Certificate[0])
		if err != nil {
			return nil, nil, false, err
		}

		// check if srv is signed by ca
		if _, err := srvCert.Verify(x509.VerifyOptions{
			Roots: roots,
		}); err == nil {
			logging.Info().Msg("OK")
			return ca, srv, false, nil
		}
		logging.Error().Msg("Agent cert and CA cert mismatch, regenerating")
	}

	// Create the CA's certificate
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"GoDoxy"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1000, 0, 0), // 1000 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, false, err
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, false, err
	}

	certPEM, keyPEM, err := saveCerts(caCertDER, caKey, caCertPath, caKeyPath)
	if err != nil {
		return nil, nil, false, err
	}

	caCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, false, err
	}

	ca = &caCert

	// Generate a new private key for the server certificate
	serverKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, false, err
	}

	srvTemplate := caTemplate
	srvTemplate.Issuer = srvTemplate.Subject
	srvTemplate.DNSNames = append(srvTemplate.DNSNames, CertsDNSName)

	srvCertDER, err := x509.CreateCertificate(rand.Reader, &srvTemplate, &caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, false, err
	}

	certPEM, keyPEM, err = saveCerts(srvCertDER, serverKey, srvCertPath, srvKeyPath)
	if err != nil {
		return nil, nil, false, err
	}

	agentCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, false, err
	}

	srv = &agentCert

	return ca, srv, true, nil
}

func NewClientCert(ca *tls.Certificate) ([]byte, []byte, error) {
	// Generate the SSL's private key
	sslKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create the SSL's certificate
	sslTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"GoDoxy"},
			CommonName:   CertsDNSName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1000, 0, 0), // 1000 years
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Sign the certificate with the CA
	sslCertDER, err := x509.CreateCertificate(rand.Reader, sslTemplate, ca.Leaf, &sslKey.PublicKey, ca.PrivateKey)
	if err != nil {
		return nil, nil, err
	}

	return saveCerts(sslCertDER, sslKey, "", "")
}
