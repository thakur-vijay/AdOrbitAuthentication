package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func newTestJWTService(t *testing.T) *Service {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	return NewService(key, &key.PublicKey, nil, "test-issuer")
}

func signTestClaims(t *testing.T, service *Service, method jwtlib.SigningMethod, claims Claims) string {
	t.Helper()
	token := jwtlib.NewWithClaims(method, claims)
	var key any = service.privateKey
	if method == jwtlib.SigningMethodHS256 {
		key = []byte("not-the-RSA-key")
	}
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return signed
}

func claimsFor(service *Service) Claims {
	now := time.Now().UTC()
	return Claims{
		UserID: "user-123",
		Role:   "advertiser",
		RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer:    service.issuer,
			Subject:   "user-123",
			ID:        "token-123",
			IssuedAt:  jwtlib.NewNumericDate(now),
			NotBefore: jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(time.Minute)),
		},
	}
}

func TestValidateAccessTokenChecksSignatureClaimsAndTime(t *testing.T) {
	service := newTestJWTService(t)
	valid := signTestClaims(t, service, jwtlib.SigningMethodRS256, claimsFor(service))

	principal, err := service.ValidateAccessToken(valid)
	if err != nil {
		t.Fatalf("ValidateAccessToken(valid) error = %v", err)
	}
	if principal.UserID != "user-123" || principal.Role != "advertiser" {
		t.Fatalf("principal = %#v, want user-123 advertiser", principal)
	}

	tampered := tamperSignature(t, valid)
	tests := []struct {
		name  string
		token string
	}{
		{name: "missing", token: ""},
		{name: "malformed", token: "not-a-jwt"},
		{name: "tampered signature", token: tampered},
		{name: "wrong signing algorithm", token: signTestClaims(t, service, jwtlib.SigningMethodHS256, claimsFor(service))},
	}

	wrongIssuer := claimsFor(service)
	wrongIssuer.Issuer = "different-issuer"
	tests = append(tests, struct {
		name  string
		token string
	}{"wrong issuer", signTestClaims(t, service, jwtlib.SigningMethodRS256, wrongIssuer)})

	wrongSubject := claimsFor(service)
	wrongSubject.Subject = "somebody-else"
	tests = append(tests, struct {
		name  string
		token string
	}{"subject does not match user", signTestClaims(t, service, jwtlib.SigningMethodRS256, wrongSubject)})

	missingUser := claimsFor(service)
	missingUser.UserID = ""
	tests = append(tests, struct {
		name  string
		token string
	}{"missing user claim", signTestClaims(t, service, jwtlib.SigningMethodRS256, missingUser)})

	missingRole := claimsFor(service)
	missingRole.Role = ""
	tests = append(tests, struct {
		name  string
		token string
	}{"missing role claim", signTestClaims(t, service, jwtlib.SigningMethodRS256, missingRole)})

	noExpiration := claimsFor(service)
	noExpiration.ExpiresAt = nil
	tests = append(tests, struct {
		name  string
		token string
	}{"missing expiration", signTestClaims(t, service, jwtlib.SigningMethodRS256, noExpiration)})

	expired := claimsFor(service)
	expired.ExpiresAt = jwtlib.NewNumericDate(time.Now().Add(-time.Minute))
	tests = append(tests, struct {
		name  string
		token string
	}{"expired", signTestClaims(t, service, jwtlib.SigningMethodRS256, expired)})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := service.ValidateAccessToken(tc.token)
			if err == nil || got != nil {
				t.Fatalf("ValidateAccessToken() = (%#v, %v), want (nil, error)", got, err)
			}
		})
	}
}

func TestAccessTokenExpiryLeeway(t *testing.T) {
	service := newTestJWTService(t)
	claims := claimsFor(service)
	claims.ExpiresAt = jwtlib.NewNumericDate(time.Now().Add(-10 * time.Second))
	if _, err := service.ValidateAccessToken(signTestClaims(t, service, jwtlib.SigningMethodRS256, claims)); err != nil {
		t.Fatalf("token within configured 30 second leeway rejected: %v", err)
	}
	claims.ExpiresAt = jwtlib.NewNumericDate(time.Now().Add(-time.Minute))
	if _, err := service.ValidateAccessToken(signTestClaims(t, service, jwtlib.SigningMethodRS256, claims)); err == nil {
		t.Fatal("token beyond configured expiry leeway was accepted")
	}
}

func TestRegistrationTokenRoundTripAndTampering(t *testing.T) {
	service := newTestJWTService(t)
	token, flowID, err := service.CreateRegistrationToken()
	if err != nil {
		t.Fatalf("CreateRegistrationToken() error = %v", err)
	}
	got, err := service.ValidateRegistrationToken(token)
	if err != nil || got != flowID {
		t.Fatalf("ValidateRegistrationToken() = (%q, %v), want flow %q", got, err, flowID)
	}
	if _, err := service.ValidateRegistrationToken(""); err == nil {
		t.Fatal("empty registration token was accepted")
	}
	mutated := tamperSignature(t, token)
	if _, err := service.ValidateRegistrationToken(mutated); err == nil {
		t.Fatal("tampered registration token was accepted")
	}
}

func tamperSignature(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[2] == "" {
		t.Fatalf("test token has invalid compact format: %q", token)
	}
	signature := []byte(parts[2])
	if signature[0] == 'A' {
		signature[0] = 'B'
	} else {
		signature[0] = 'A'
	}
	return parts[0] + "." + parts[1] + "." + string(signature)
}

func TestLoadRSAKeyPEMFormatsAndErrors(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	dir := t.TempDir()

	writePEM := func(name, blockType string, der []byte) string {
		t.Helper()
		path := filepath.Join(dir, name)
		data := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
		return path
	}

	pkcs8Path := writePEM("private-pkcs8.pem", "PRIVATE KEY", mustMarshalPKCS8(t, key))
	pkcs1Path := writePEM("private-pkcs1.pem", "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key))
	publicPath := writePEM("public.pem", "PUBLIC KEY", mustMarshalPKIX(t, &key.PublicKey))
	certificateTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, certificateTemplate, certificateTemplate, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	certificatePath := writePEM("public-cert.pem", "CERTIFICATE", certificateDER)

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "PKCS8 private key", path: pkcs8Path},
		{name: "PKCS1 private key", path: pkcs1Path},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loaded, err := LoadPrivateKey(tc.path)
			if err != nil || loaded.N.Cmp(key.N) != 0 {
				t.Fatalf("LoadPrivateKey() returned key %v, error %v", loaded, err)
			}
		})
	}
	loadedPublic, err := LoadPublicKey(publicPath)
	if err != nil || loadedPublic.N.Cmp(key.N) != 0 {
		t.Fatalf("LoadPublicKey() returned key %v, error %v", loadedPublic, err)
	}
	loadedCertificateKey, err := LoadPublicKey(certificatePath)
	if err != nil || loadedCertificateKey.N.Cmp(key.N) != 0 {
		t.Fatalf("LoadPublicKey(certificate) returned key %v, error %v", loadedCertificateKey, err)
	}

	badPEM := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(badPEM, []byte("not PEM"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPrivateKey(filepath.Join(dir, "missing.pem")); err == nil {
		t.Fatal("LoadPrivateKey() accepted a missing file")
	}
	if _, err := LoadPrivateKey(badPEM); err == nil || !strings.Contains(err.Error(), "PEM") {
		t.Fatalf("LoadPrivateKey() malformed PEM error = %v", err)
	}
	if _, err := LoadPublicKey(badPEM); err == nil || !strings.Contains(err.Error(), "PEM") {
		t.Fatalf("LoadPublicKey() malformed PEM error = %v", err)
	}
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateECDSAKey() error = %v", err)
	}
	ecdsaPrivateDER, err := x509.MarshalPKCS8PrivateKey(ecdsaKey)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey(ECDSA) error = %v", err)
	}
	ecdsaPrivatePath := writePEM("non-rsa-private.pem", "PRIVATE KEY", ecdsaPrivateDER)
	if _, err := LoadPrivateKey(ecdsaPrivatePath); err == nil || !strings.Contains(err.Error(), "not RSA") {
		t.Fatalf("LoadPrivateKey(ECDSA) error = %v, want non-RSA error", err)
	}
	ecdsaPublicDER, err := x509.MarshalPKIXPublicKey(&ecdsaKey.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey(ECDSA) error = %v", err)
	}
	wrongPublicPath := writePEM("non-rsa-public.pem", "PUBLIC KEY", ecdsaPublicDER)
	if _, err := LoadPublicKey(wrongPublicPath); err == nil || !strings.Contains(err.Error(), "not RSA") {
		t.Fatalf("LoadPublicKey(ECDSA) error = %v, want non-RSA error", err)
	}
	wrongType := writePEM("public-as-private.pem", "PUBLIC KEY", mustMarshalPKIX(t, &key.PublicKey))
	if _, err := LoadPrivateKey(wrongType); err == nil {
		t.Fatal("LoadPrivateKey() accepted a public key")
	}
}

func mustMarshalPKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey() error = %v", err)
	}
	return der
}

func mustMarshalPKIX(t *testing.T, key *rsa.PublicKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey() error = %v", err)
	}
	return der
}

func BenchmarkGenerateAndValidateAccessToken(b *testing.B) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal(err)
	}
	service := NewService(key, &key.PublicKey, nil, "benchmark-issuer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token, err := service.generateAccessToken("user-id", "advertiser")
		if err != nil {
			b.Fatal(err)
		}
		if _, err := service.ValidateAccessToken(token); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateAccessToken(b *testing.B) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal(err)
	}
	service := NewService(key, &key.PublicKey, nil, "benchmark-issuer")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.generateAccessToken("user-id", "advertiser"); err != nil {
			b.Fatal(err)
		}
	}
}
