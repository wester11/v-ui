// Package mtls — внутренний CA, выпускающий клиентские сертификаты для агентов.
//
// CA-ключ хранится на диске рядом с .env (по дефолту /opt/void-wg/runtime/agent-ca/).
// Срок жизни клиентского сертификата — 5 лет. Агент использует его для
// аутентификации перед control-plane (mTLS), control-plane — для проверки.
package mtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	caCertFile = "ca.crt"
	caKeyFile  = "ca.key"
	caCN       = "void-wg agent CA"
)

type Issuer struct {
	dir   string
	mu    sync.Mutex
	caCrt *x509.Certificate
	caKey *ecdsa.PrivateKey
	caPEM []byte
}

func NewIssuer(dir string) (*Issuer, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	is := &Issuer{dir: dir}
	if err := is.loadOrCreate(); err != nil {
		return nil, err
	}
	return is, nil
}

func (is *Issuer) loadOrCreate() error {
	is.mu.Lock()
	defer is.mu.Unlock()

	cp := filepath.Join(is.dir, caCertFile)
	kp := filepath.Join(is.dir, caKeyFile)

	if cb, err := os.ReadFile(cp); err == nil {
		if kb, err := os.ReadFile(kp); err == nil {
			cert, key, err := decodeCA(cb, kb)
			if err == nil {
				is.caCrt, is.caKey, is.caPEM = cert, key, cb
				return nil
			}
		}
	}

	// Создаём новый CA.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: caCN},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(20, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &priv.PublicKey, priv)
	if err != nil {
		return err
	}
	caCert, err := x509.ParseCertificate(der)
	if err != nil {
		return err
	}

	caCrtPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(cp, caCrtPEM, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(kp, caKeyPEM, 0o600); err != nil {
		return err
	}

	is.caCrt = caCert
	is.caKey = priv
	is.caPEM = caCrtPEM
	return nil
}

// IssueAgentCert — выпускает клиентский cert для агента.
// Возвращает PEM (caBundle, cert, key) и SHA-256 fingerprint cert'а в hex.
func (is *Issuer) IssueAgentCert(serverID uuid.UUID) ([]byte, []byte, []byte, string, error) {
	is.mu.Lock()
	defer is.mu.Unlock()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, "", err
	}

	// SerialNumber — uuid в виде 16-байтового big-int.
	sn := new(big.Int).SetBytes(serverID[:])
	tpl := &x509.Certificate{
		SerialNumber: sn,
		Subject:      pkix.Name{CommonName: "void-wg-agent/" + serverID.String()},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().AddDate(5, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, is.caCrt, &priv.PublicKey, is.caKey)
	if err != nil {
		return nil, nil, nil, "", err
	}
	fp := sha256.Sum256(der)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, nil, "", err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return is.caPEM, certPEM, keyPEM, hex.EncodeToString(fp[:]), nil
}

// CABundle — публичный CA для проверки агентских коннекций.
func (is *Issuer) CABundle() []byte { return is.caPEM }

func decodeCA(crtPEM, keyPEM []byte) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	cb, _ := pem.Decode(crtPEM)
	if cb == nil {
		return nil, nil, fmt.Errorf("ca cert: pem decode failed")
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, nil, err
	}
	kb, _ := pem.Decode(keyPEM)
	if kb == nil {
		return nil, nil, fmt.Errorf("ca key: pem decode failed")
	}
	key, err := x509.ParseECPrivateKey(kb.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}
