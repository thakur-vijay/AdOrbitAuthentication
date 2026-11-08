package emailotp

import (
	"regexp"
	"testing"
)

func TestGenerateOTPProducesSixDigitCode(t *testing.T) {
	for i := 0; i < 100; i++ {
		otp, err := generateOTP()
		if err != nil {
			t.Fatalf("generateOTP() error = %v", err)
		}
		if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(otp) {
			t.Fatalf("generateOTP() = %q, want six ASCII digits", otp)
		}
		if otp < "100000" || otp > "999999" {
			t.Fatalf("generateOTP() = %q, outside the configured range", otp)
		}
	}
}

func TestOTPHMACBindsCodeToUserAndComparesConstantTime(t *testing.T) {
	service := NewService(nil, nil, []byte("test-only-secret"))
	hash := service.generateOTPHMAC("user-a", "012345")
	if !service.verifyOTPHMAC("user-a", "012345", hash) {
		t.Fatal("verifyOTPHMAC() rejected the matching user and code")
	}
	if service.verifyOTPHMAC("user-b", "012345", hash) {
		t.Fatal("verifyOTPHMAC() accepted a code under a different user ID")
	}
	if service.verifyOTPHMAC("user-a", "012346", hash) {
		t.Fatal("verifyOTPHMAC() accepted a different code")
	}
}

func BenchmarkGenerateOTP(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := generateOTP(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOTPHMAC(b *testing.B) {
	service := NewService(nil, nil, []byte("benchmark-secret"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.generateOTPHMAC("user-id", "123456")
	}
}
