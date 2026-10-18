// All credit to minica for this: https://github.com/jsha/minica
//
// I picked apart that code to figure out how to do this in here, mainly because
// minica isn't really built in a way that I could pull it in as a package.
//
// TODO: Fork the library and make it more of a package to import, then submit
// back.
package tls

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/adamkirk/orca/internal/bus"
	"github.com/adamkirk/orca/internal/usererr"
	"github.com/adamkirk/orca/internal/workspace"
	"github.com/spf13/afero"
)

type tui interface {
	Info(msg ...string)
	Success(msg ...string)
	NewLine()
}

type issuer struct {
	key  crypto.Signer
	cert *x509.Certificate
}

type Manager struct {
	fs     afero.Fs
	tui    tui
	tlsDir string
}

func (m *Manager) GetCertsDir() string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(m.tlsDir, "/"), "certs")
}

func (m *Manager) GetRootKeyPath() string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(m.tlsDir, "/"), "key.pem")
}

func (m *Manager) GetRootCertPath() string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(m.tlsDir, "/"), "cert.pem")
}

func (m *Manager) createKey(path string) (crypto.Signer, error) {
	var key crypto.Signer
	var err error

	key, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

	if err != nil {
		return nil, err
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)

	if err != nil {
		return nil, err
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})

	if err := afero.WriteFile(m.fs, path, b, 0600); err != nil {
		return nil, err
	}

	return key, nil
}

func (m *Manager) calculateSKID(pubKey crypto.PublicKey) ([]byte, error) {
	spkiASN1, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	var spki struct {
		Algorithm        pkix.AlgorithmIdentifier
		SubjectPublicKey asn1.BitString
	}
	_, err = asn1.Unmarshal(spkiASN1, &spki)
	if err != nil {
		return nil, err
	}
	skid := sha1.Sum(spki.SubjectPublicKey.Bytes)
	return skid[:], nil
}

func (m *Manager) createRootCert(key crypto.Signer, filename string) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))

	if err != nil {
		return nil, err
	}

	skid, err := m.calculateSKID(key.Public())

	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "orca root ca " + hex.EncodeToString(serial.Bytes()[:3]),
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(100, 0, 0),

		SubjectKeyId:          skid,
		AuthorityKeyId:        skid,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)

	if err != nil {
		return nil, err
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})

	if err := afero.WriteFile(m.fs, filename, b, 0600); err != nil {
		return nil, err
	}

	return x509.ParseCertificate(der)
}

func (m *Manager) readCert(path string) (*x509.Certificate, error) {
	contents, err := afero.ReadFile(m.fs, path)

	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(contents)
	if block == nil {
		return nil, fmt.Errorf("no PEM found")
	} else if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("incorrect PEM type %s", block.Type)
	}

	return x509.ParseCertificate(block.Bytes)
}

func (m *Manager) readKey(path string) (crypto.Signer, error) {
	contents, err := afero.ReadFile(m.fs, path)

	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(contents)
	if block == nil {
		return nil, fmt.Errorf("no PEM found")
	} else if block.Type == "PRIVATE KEY" {
		signer, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8: %w", err)
		}

		return signer.(*ecdsa.PrivateKey), nil
	}

	return nil, fmt.Errorf("incorrect PEM type %s", block.Type)
}

func (m *Manager) getOrCreateKey(path string) (crypto.Signer, error) {
	exists, err := afero.Exists(m.fs, path)

	if err != nil {
		return nil, err
	}

	if !exists {
		m.tui.Info(fmt.Sprintf("Creating key: %s", path))
		key, err := m.createKey(path)

		if err != nil {
			return nil, usererr.Wrap(err, fmt.Sprintf("Failed to create key %s", path))
		}

		m.tui.Success(fmt.Sprintf("Key %s created successfully!", path))
		return key, nil
	}

	m.tui.Info(fmt.Sprintf("Key %s already exists, skipping creation.", path))

	cert, err := m.readKey(path)

	return cert, usererr.Wrap(err, fmt.Sprintf("Failed to load key %s", path))
}

func (m *Manager) getOrCreateRootCert(key crypto.Signer) (*x509.Certificate, error) {
	path := m.GetRootCertPath()
	exists, err := afero.Exists(m.fs, path)

	if err != nil {
		return nil, err
	}

	if !exists {
		m.tui.Info(fmt.Sprintf("Creating certificate: %s", path))

		cert, err := m.createRootCert(key, path)

		if err != nil {
			return nil, usererr.Wrap(err, fmt.Sprintf("Failed to create certificate %s", path))
		}

		m.tui.Success(fmt.Sprintf("Certificate %s created successfully!", path))
		return cert, nil
	}

	m.tui.Info(fmt.Sprintf("Certificate %s already exists, skipping creation.", path))

	return m.readCert(path)
}

func (m *Manager) getRootIssuer() (*issuer, error) {
	key, err := m.getOrCreateKey(m.GetRootKeyPath())

	if err != nil {
		return nil, err
	}

	cert, err := m.getOrCreateRootCert(key)

	if err != nil {
		return nil, err
	}

	return &issuer{
		key:  key,
		cert: cert,
	}, nil
}

func (m *Manager) createTLSDirsIfMissing() error {
	exists, err := afero.DirExists(m.fs, m.GetCertsDir())

	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return m.fs.MkdirAll(m.GetCertsDir(), 0700)
}

func (m *Manager) generateCertificateIfNotPresent(issuer *issuer, domain string) error {

	cn := domain
	var certName = strings.Replace(cn, "*", "_", -1)

	keyPath := fmt.Sprintf("%s/%s.key", m.GetCertsDir(), certName)

	key, err := m.getOrCreateKey(keyPath)

	if err != nil {
		return err
	}

	certPath := fmt.Sprintf("%s/%s.cert", m.GetCertsDir(), certName)

	returnErrMessage := func(err error) error {
		return usererr.Wrap(err, fmt.Sprintf("Failed to create certificate %s", certPath))
	}

	certExists, err := afero.Exists(m.fs, certPath)

	if err != nil {
		return err
	}

	if certExists {
		m.tui.Info(fmt.Sprintf("Certificate %s already exists, skipping creation.", certPath))
		return nil
	}

	m.tui.Info(fmt.Sprintf("Creating certificate: %s", certPath))
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))

	if err != nil {
		return returnErrMessage(err)
	}

	template := &x509.Certificate{
		DNSNames:    []string{domain},
		IPAddresses: []net.IP{},
		Subject: pkix.Name{
			CommonName: domain,
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		// Set the validity period to 2 years and 30 days, to satisfy the iOS and
		// macOS requirements that all server certificates must have validity
		// shorter than 825 days:
		// https://derflounder.wordpress.com/2019/06/06/new-tls-security-requirements-for-ios-13-and-macos-catalina-10-15/
		NotAfter: time.Now().AddDate(2, 0, 30),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, issuer.cert, key.Public(), issuer.key)

	if err != nil {
		return returnErrMessage(err)
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})

	if err := afero.WriteFile(m.fs, certPath, b, 0600); err != nil {
		return returnErrMessage(err)
	}

	m.tui.Success(fmt.Sprintf("Certificate %s created successfully!", certPath))
	return nil
}

type GenerateCommand struct {
	bus.Returns[bus.Void]

	Workspace string
}

func (GenerateCommand) MessageName() string {
	return "tls.gen"
}

func (m *Manager) Generate(ctx context.Context, cmd GenerateCommand) error {
	execCtx, err := workspace.ExecutionContextFrom(ctx)

	if err != nil {
		return err
	}

	m.tui.Info("Generating Root certificate if required...")
	if err := m.createTLSDirsIfMissing(); err != nil {
		return usererr.Wrap(err, fmt.Sprintf("Failed to create certificates directory at: %s", m.GetCertsDir()))
	}

	issuer, err := m.getRootIssuer()
	if err != nil {
		return err
	}

	certsToGenerate := execCtx.Workspace.GetUniqueTLSCertificates()

	for _, c := range certsToGenerate {
		m.tui.NewLine()
		m.tui.Info(fmt.Sprintf("Generating certificate for '%s'...", c))
		if err := m.generateCertificateIfNotPresent(issuer, c); err != nil {
			return err
		}
	}

	return nil
}

func NewManager(fs afero.Fs, tui tui, tlsDir string) *Manager {
	return &Manager{
		fs:     fs,
		tlsDir: tlsDir,
		tui:    tui,
	}
}
