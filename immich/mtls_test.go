package immich_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/simulot/immich-go/immich"
)

// certPEM holds a PEM-encoded certificate and its private key.
type certPEM struct {
	cert []byte
	key  []byte
	tls  tls.Certificate
}

// makeCert issues a certificate signed by the given parent (or self-signed when
// parent is nil). It returns the PEM material and a usable tls.Certificate.
func makeCert(t *testing.T, cn string, isCA bool, parent *x509.Certificate, parentKey *ecdsa.PrivateKey) (certPEM, *x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:              []string{"127.0.0.1", "localhost"},
		IsCA:                  isCA,
		BasicConstraintsValid: true,
	}

	signer, signerKey := tmpl, key
	if parent != nil {
		signer, signerKey = parent, parentKey
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, signer, &key.PublicKey, signerKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certOut := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyOut := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	tlsCert, err := tls.X509KeyPair(certOut, keyOut)
	if err != nil {
		t.Fatalf("build tls keypair: %v", err)
	}

	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	return certPEM{cert: certOut, key: keyOut, tls: tlsCert}, parsed, key
}

// TestMutualTLS stands up a TLS server that requires and verifies a client
// certificate, then confirms the client authenticates successfully when
// configured with a client certificate and the trust anchor, and fails when
// the client certificate is omitted.
func TestMutualTLS(t *testing.T) {
	_, caCert, caKey := makeCert(t, "immich-go-test-ca", true, nil, nil)
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})

	serverCert, _, _ := makeCert(t, "127.0.0.1", false, caCert, caKey)
	clientCert, _, _ := makeCert(t, "immich-go-client", false, caCert, caKey)

	clientCAs := x509.NewCertPool()
	clientCAs.AppendCertsFromPEM(caPEM)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"res":"pong"}`))
	}))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert.tls},
		ClientCAs:    clientCAs,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	server.StartTLS()
	defer server.Close()

	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	certFile := filepath.Join(dir, "client.pem")
	keyFile := filepath.Join(dir, "client.key")
	writeFile(t, caFile, caPEM)
	writeFile(t, certFile, clientCert.cert)
	writeFile(t, keyFile, clientCert.key)

	t.Run("with client certificate", func(t *testing.T) {
		client, err := immich.NewImmichClient(server.URL, "test-key",
			immich.OptionCACertificate(caFile),
			immich.OptionClientCertificate(certFile, keyFile),
		)
		if err != nil {
			t.Fatalf("NewImmichClient: %v", err)
		}
		if err := client.PingServer(context.Background()); err != nil {
			t.Fatalf("expected mTLS ping to succeed, got %v", err)
		}
	})

	t.Run("without client certificate", func(t *testing.T) {
		client, err := immich.NewImmichClient(server.URL, "test-key",
			immich.OptionCACertificate(caFile),
		)
		if err != nil {
			t.Fatalf("NewImmichClient: %v", err)
		}
		if err := client.PingServer(context.Background()); err == nil {
			t.Fatal("expected ping to fail without a client certificate, got nil error")
		}
	})

	t.Run("mismatched cert and key", func(t *testing.T) {
		_, err := immich.NewImmichClient(server.URL, "test-key",
			immich.OptionClientCertificate(certFile, ""),
		)
		if err == nil {
			t.Fatal("expected error when only one of cert/key is supplied")
		}
	})
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
