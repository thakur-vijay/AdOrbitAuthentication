//go:build integration

package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/adorbit/authentication/internal/messaging"
)

func TestRabbitMQCreateAndFetchRPCProtocol(t *testing.T) {
	brokerURL := requireEnv(t, "TEST_RABBITMQ_URL")
	harness := startUsersServiceHarness(t)
	client, err := messaging.NewRabbitMQ(brokerURL)
	if err != nil {
		t.Fatalf("NewRabbitMQ() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close RabbitMQ client: %v", err)
		}
	})

	userID := mustUUID(t)
	created, err := client.CreateUser(context.Background(), messaging.CreateUserRequest{
		UserID:   userID,
		FullName: "MISMATCH_CORRELATION",
		Email:    "rabbit@example.com",
		Role:     "publisher",
	})
	if err != nil {
		t.Fatalf("CreateUser() with mismatched then matching reply error = %v", err)
	}
	if created.UserID != userID || created.Email != "rabbit@example.com" || created.Role != "publisher" {
		t.Fatalf("CreateUser() = %#v, want matching persisted user", created)
	}

	fetched, err := client.FetchUser(context.Background(), messaging.FetchUserRequest{UserID: userID, Role: "publisher"})
	if err != nil {
		t.Fatalf("FetchUser() error = %v", err)
	}
	if fetched.UserID != userID || fetched.Role != "publisher" || fetched.Email != "rabbit@example.com" {
		t.Fatalf("FetchUser() = %#v, want stored user identity and role", fetched)
	}

	if _, err := client.CreateUser(context.Background(), messaging.CreateUserRequest{UserID: userID, Role: "publisher"}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate CreateUser() error = %v, want duplicate-user conflict", err)
	}
	if _, err := client.FetchUser(context.Background(), messaging.FetchUserRequest{UserID: mustUUID(t), Role: "publisher"}); err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("FetchUser(missing) error = %v, want user-not-found", err)
	}
	if _, err := client.FetchUser(context.Background(), messaging.FetchUserRequest{UserID: userID, Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "unexpected user role") {
		t.Fatalf("FetchUser(wrong role) error = %v, want role mismatch", err)
	}

	if _, err := client.CreateUser(context.Background(), messaging.CreateUserRequest{UserID: mustUUID(t), FullName: "UNEXPECTED_ID", Role: "publisher"}); err == nil || !strings.Contains(err.Error(), "unexpected user ID") {
		t.Fatalf("CreateUser(unexpected ID) error = %v", err)
	}
	if _, err := client.CreateUser(context.Background(), messaging.CreateUserRequest{UserID: mustUUID(t), FullName: "MALFORMED_RESPONSE", Role: "publisher"}); err == nil || !strings.Contains(err.Error(), "Invalid response") {
		t.Fatalf("CreateUser(malformed response) error = %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.FetchUser(cancelled, messaging.FetchUserRequest{UserID: userID, Role: "publisher"}); err == nil || !strings.Contains(err.Error(), "cancel") {
		t.Fatalf("FetchUser(cancelled context) error = %v", err)
	}

	timedOut, timeoutCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer timeoutCancel()
	result := make(chan error, 1)
	go func() {
		_, err := client.CreateUser(timedOut, messaging.CreateUserRequest{UserID: mustUUID(t), FullName: "HOLD_RESPONSE", Role: "advertiser"})
		result <- err
	}()
	select {
	case <-harness.held:
	case <-time.After(3 * time.Second):
		t.Fatal("test Users Service did not receive the timeout request")
	}
	if err := <-result; !errors.Is(err, context.DeadlineExceeded) && (err == nil || !strings.Contains(err.Error(), "Timed out waiting")) {
		t.Fatalf("CreateUser(timeout) error = %v, want bounded timeout error", err)
	}
}
