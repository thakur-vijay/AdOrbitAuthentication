package messaging

import (
	"context"
	"strings"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRabbitMQOperationsRejectMissingConnectionAndInvalidRequests(t *testing.T) {
	var absent *RabbitMQ
	if _, err := absent.CreateUser(context.Background(), CreateUserRequest{UserID: "user", Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("CreateUser(nil connection) error = %v, want connection initialization error", err)
	}
	if _, err := absent.FetchUser(context.Background(), FetchUserRequest{UserID: "user", Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("FetchUser(nil connection) error = %v, want connection initialization error", err)
	}

	initialized := &RabbitMQ{Connection: &amqp.Connection{}}
	for _, tc := range []struct {
		name string
		run  func(context.Context) error
		want string
	}{
		{
			name: "create requires user ID",
			run: func(ctx context.Context) error {
				_, err := initialized.CreateUser(ctx, CreateUserRequest{Role: "advertiser"})
				return err
			},
			want: "User ID is required",
		},
		{
			name: "create requires role",
			run: func(ctx context.Context) error {
				_, err := initialized.CreateUser(ctx, CreateUserRequest{UserID: "user"})
				return err
			},
			want: "Role is required",
		},
		{
			name: "fetch requires user ID",
			run: func(ctx context.Context) error {
				_, err := initialized.FetchUser(ctx, FetchUserRequest{Role: "advertiser"})
				return err
			},
			want: "User ID is required",
		},
		{
			name: "fetch requires role",
			run: func(ctx context.Context) error {
				_, err := initialized.FetchUser(ctx, FetchUserRequest{UserID: "user"})
				return err
			},
			want: "Role is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(context.Background()); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want message containing %q", err, tc.want)
			}
		})
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := initialized.CreateUser(cancelled, CreateUserRequest{UserID: "user", Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "cancel") {
		t.Fatalf("CreateUser(cancelled context) error = %v", err)
	}
	if _, err := initialized.FetchUser(cancelled, FetchUserRequest{UserID: "user", Role: "advertiser"}); err == nil || !strings.Contains(err.Error(), "cancel") {
		t.Fatalf("FetchUser(cancelled context) error = %v", err)
	}

	if err := absent.Close(); err != nil {
		t.Fatalf("Close() on nil receiver error = %v", err)
	}
	if _, err := NewRabbitMQ("not a valid AMQP URL"); err == nil {
		t.Fatal("NewRabbitMQ() accepted an invalid URL")
	}
}
