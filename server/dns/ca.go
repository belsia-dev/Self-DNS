package dns

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	caCertFile = "blockpage-ca-cert.pem"
	caKeyFile  = "blockpage-ca-key.pem"
)

type blockPageCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey

	mu    sync.Mutex
	cache map[string]*tls.Certificate
}

func loadOrCreateCA(dir string) (*blockPageCA, error) {
	if dir == "" {
		return generateCA()
	}

	certPEM, certErr := os.ReadFile(filepath.Join(dir, caCertFile))
	keyPEM, keyErr := os.ReadFile(filepath.Join(dir, caKeyFile))
	if certErr == nil && keyErr == nil {
		if ca, err := parseCA(certPEM, keyPEM); err == nil {
			return ca, nil
		}
	}

	ca, err := generateCA()
	if err != nil {
		return nil, err
	}

	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, caCertFile), ca.CertPEM(), 0o644)
	keyDER, _ := x509.MarshalECPrivateKey(ca.key)
	_ = os.WriteFile(filepath.Join(dir, caKeyFile),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
		0o600,
	)
	return ca, nil
}

func generateCA() (*blockPageCA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate CA serial: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "SelfDNS Local CA", Organization: []string{"SelfDNS"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create CA cert: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}
	return &blockPageCA{cert: cert, key: key, cache: make(map[string]*tls.Certificate)}, nil
}

func parseCA(certPEM, keyPEM []byte) (*blockPageCA, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in cert file")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse cert: %w", err)
	}
	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in key file")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}
	return &blockPageCA{cert: cert, key: key, cache: make(map[string]*tls.Certificate)}, nil
}

func (ca *blockPageCA) CertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw})
}

func (ca *blockPageCA) tlsConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: ca.getCertificate,
		MinVersion:     tls.VersionTLS12,
	}
}

func (ca *blockPageCA) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	name := hello.ServerName
	if name == "" {
		name = "blocked.local"
	}

	ca.mu.Lock()
	if c, ok := ca.cache[name]; ok {
		ca.mu.Unlock()
		return c, nil
	}
	ca.mu.Unlock()

	c, err := ca.issue(name)
	if err != nil {
		return nil, err
	}

	ca.mu.Lock()
	ca.cache[name] = c
	ca.mu.Unlock()
	return c, nil
}

func (ca *blockPageCA) issue(hostname string) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	const maxLeafValidity = 397 * 24 * time.Hour
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: hostname},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(maxLeafValidity),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(hostname); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{hostname}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		return nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}
