package mongodb

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// generateSelfSignedCertPEM returns a combined certificate+private-key PEM, the same shape a
// user would pass as certificate_key_file (concatenating both blocks into one value).
func generateSelfSignedCertPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating key: %s", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating certificate: %s", err)
	}

	var out []byte
	out = append(out, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	out = append(out, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})...)
	return out
}

func testDialer(t *testing.T) options.ContextDialer {
	t.Helper()
	d, err := proxyDialer(&ClientConfig{})
	if err != nil {
		t.Fatalf("building dialer: %s", err)
	}
	return d
}

func TestClientOptionsUsernamePasswordAuth(t *testing.T) {
	c := &ClientConfig{Host: "localhost", Port: "27017", DB: "admin", Username: "root", Password: "secret"}
	opts, err := c.clientOptions(testDialer(t))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if opts.Auth == nil {
		t.Fatal("Auth credential was not set")
	}
	if opts.Auth.AuthMechanism != "" {
		t.Fatalf("AuthMechanism = %q, want empty (default SCRAM) for username/password auth", opts.Auth.AuthMechanism)
	}
	if opts.Auth.Username != "root" || opts.Auth.Password != "secret" || opts.Auth.AuthSource != "admin" {
		t.Fatalf("credential = %+v, want username/password/authSource carried through from ClientConfig", opts.Auth)
	}
	if opts.TLSConfig != nil {
		t.Fatal("TLSConfig should be nil when no certificate/certificate_key_file is configured")
	}
}

// This is the exact regression PR #35 introduced: an X.509 credential built on one
// *options.ClientOptions and discarded, while a second, separate one (without it) was what
// actually got passed to mongo.NewClient. Asserting against clientOptions()'s own return value
// closes that gap - there's no second object here for the credential to have gone missing on.
func TestClientOptionsX509Auth(t *testing.T) {
	certKeyPEM := generateSelfSignedCertPEM(t)
	c := &ClientConfig{Host: "localhost", Port: "27017", CertificateKeyFile: string(certKeyPEM)}
	opts, err := c.clientOptions(testDialer(t))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if opts.Auth == nil {
		t.Fatal("Auth credential was not set")
	}
	if opts.Auth.AuthMechanism != "MONGODB-X509" {
		t.Fatalf("AuthMechanism = %q, want MONGODB-X509", opts.Auth.AuthMechanism)
	}
	if opts.Auth.Username != "" || opts.Auth.Password != "" {
		t.Fatalf("credential = %+v, want no username/password sent for X.509 auth", opts.Auth)
	}
	if opts.TLSConfig == nil {
		t.Fatal("TLSConfig was not set for X.509 client-certificate auth")
	}
	if len(opts.TLSConfig.Certificates) != 1 {
		t.Fatalf("TLSConfig.Certificates has %d entries, want 1 (the parsed client cert/key)", len(opts.TLSConfig.Certificates))
	}
}

func TestClientOptionsX509AuthTakesPrecedenceOverUsernamePassword(t *testing.T) {
	certKeyPEM := generateSelfSignedCertPEM(t)
	c := &ClientConfig{Host: "localhost", Port: "27017", Username: "root", Password: "secret", CertificateKeyFile: string(certKeyPEM)}
	opts, err := c.clientOptions(testDialer(t))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if opts.Auth.AuthMechanism != "MONGODB-X509" {
		t.Fatalf("AuthMechanism = %q, want MONGODB-X509 to take precedence when certificate_key_file is set", opts.Auth.AuthMechanism)
	}
}

func TestGetTLSConfigWithAllServerCertificatesCAOnly(t *testing.T) {
	// A CA bundle with two certs - the exact shape (e.g. AWS's rds-combined-ca-bundle.pem)
	// that a driver-only tlsCAFile= URI param is documented to only read the first cert from.
	cert1 := generateSelfSignedCertPEM(t)
	cert2 := generateSelfSignedCertPEM(t)
	bundle := append(append([]byte{}, cert1...), cert2...)

	tlsConfig, err := getTLSConfigWithAllServerCertificates(bundle, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if tlsConfig.RootCAs == nil {
		t.Fatal("RootCAs not populated")
	}
	if len(tlsConfig.RootCAs.Subjects()) != 2 { //nolint:staticcheck // Subjects() is deprecated but still the simplest way to count loaded certs in a test
		t.Fatalf("RootCAs has %d subjects, want 2 (both certs in the bundle)", len(tlsConfig.RootCAs.Subjects()))
	}
	if len(tlsConfig.Certificates) != 0 {
		t.Fatalf("Certificates has %d entries, want 0 when no client cert/key was given", len(tlsConfig.Certificates))
	}
}

func TestGetTLSConfigWithAllServerCertificatesInvalidClientCert(t *testing.T) {
	if _, err := getTLSConfigWithAllServerCertificates(nil, []byte("not a valid pem"), false); err == nil {
		t.Fatal("expected an error for an invalid client certificate/key PEM, got nil")
	}
}
