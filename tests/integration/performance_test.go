//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"

	"github.com/adorbit/authentication/internal/auth"
	appjwt "github.com/adorbit/authentication/internal/jwt"
	"github.com/adorbit/authentication/internal/messaging"
	"github.com/adorbit/authentication/internal/security"
)

func BenchmarkPostgresCredentialLookup(b *testing.B) {
	db := postgresFixture(b)
	repository := auth.NewRepository(db)
	userID := mustUUID(b)
	email := fmt.Sprintf("benchmark-%s@example.com", userID)
	insertCredential(b, db, userID, email)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		credential, err := repository.FetchByEmail(context.Background(), email)
		if err != nil || credential == nil || credential.UserID != userID {
			b.Fatalf("FetchByEmail() = (%#v, %v)", credential, err)
		}
	}
}

func BenchmarkVerifiedLoginWorkflow(b *testing.B) {
	db := postgresFixture(b)
	startUsersServiceHarness(b)
	brokerURL := requireEnv(b, "TEST_RABBITMQ_URL")
	rabbit, err := messaging.NewRabbitMQ(brokerURL)
	if err != nil {
		b.Fatalf("connect to RabbitMQ: %v", err)
	}
	b.Cleanup(func() {
		if err := rabbit.Close(); err != nil {
			b.Errorf("close benchmark RabbitMQ connection: %v", err)
		}
	})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("generate benchmark JWT key: %v", err)
	}
	userID := mustUUID(b)
	const email = "login-benchmark@example.com"
	const password = "BenchmarkPass9!"
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		b.Fatalf("HashPassword() error = %v", err)
	}
	insertCredential(b, db, userID, email)
	if _, err := db.Exec(`UPDATE credentials SET password_hash = $1, status = 'active', is_email_verified = TRUE WHERE user_id = $2`, passwordHash, userID); err != nil {
		b.Fatalf("set up verified benchmark credential: %v", err)
	}
	if _, err := rabbit.CreateUser(context.Background(), messaging.CreateUserRequest{UserID: userID, FullName: "Benchmark User", Email: email, Role: "advertiser"}); err != nil {
		b.Fatalf("create benchmark user in Users Service: %v", err)
	}
	service := auth.NewService(
		appjwt.NewService(key, &key.PublicKey, appjwt.NewRepository(db), "benchmark-issuer"),
		auth.NewRepository(db),
		rabbit,
		nil,
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := service.Login(context.Background(), auth.LoginRequestDTO{Email: email, Password: password, Role: "advertiser"})
		if err != nil || result == nil || result.AccessToken == nil || result.RefreshToken == nil {
			b.Fatalf("Login() = (%#v, %v)", result, err)
		}
	}
}
