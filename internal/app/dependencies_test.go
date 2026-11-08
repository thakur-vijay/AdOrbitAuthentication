package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/adorbit/authentication/internal/config"
)

func TestParseRSAKeyPEMFormatsAndRejectsInvalidKeys(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateECDSAKey() error = %v", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		t.Fatal(err)
	}
	pkix, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	private, err := parsePrivateKey(string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})))
	if err != nil || private.N.Cmp(rsaKey.N) != 0 {
		t.Fatalf("parsePrivateKey(PKCS8) = (%v, %v)", private, err)
	}
	private, err = parsePrivateKey(string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)})))
	if err != nil || private.N.Cmp(rsaKey.N) != 0 {
		t.Fatalf("parsePrivateKey(PKCS1) = (%v, %v)", private, err)
	}
	public, err := parsePublicKey(string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix})))
	if err != nil || public.N.Cmp(rsaKey.N) != 0 {
		t.Fatalf("parsePublicKey() = (%v, %v)", public, err)
	}

	ecdsaPrivate, err := x509.MarshalPKCS8PrivateKey(ecdsaKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsePrivateKey(string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: ecdsaPrivate}))); err == nil || !strings.Contains(err.Error(), "not RSA") {
		t.Fatalf("parsePrivateKey(EC) error = %v, want non-RSA error", err)
	}
	ecdsaPublic, err := x509.MarshalPKIXPublicKey(&ecdsaKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsePublicKey(string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: ecdsaPublic}))); err == nil || !strings.Contains(err.Error(), "not RSA") {
		t.Fatalf("parsePublicKey(EC) error = %v, want non-RSA error", err)
	}
	if _, err := parsePrivateKey("not PEM"); err == nil || !strings.Contains(err.Error(), "PEM") {
		t.Fatalf("parsePrivateKey(malformed PEM) error = %v", err)
	}
	if _, err := parsePublicKey("not PEM"); err == nil || !strings.Contains(err.Error(), "PEM") {
		t.Fatalf("parsePublicKey(malformed PEM) error = %v", err)
	}

	if _, err := BuildRouter(nil, &config.Config{JWTPrivateKey: "not PEM"}, nil); err == nil || !strings.Contains(err.Error(), "private key PEM") {
		t.Fatalf("BuildRouter(invalid key) error = %v, want PEM parse error", err)
	}
}
