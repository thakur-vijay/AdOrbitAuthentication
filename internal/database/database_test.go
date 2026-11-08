package database

import (
	"context"
	"strings"
	"testing"
)

func TestConnectRejectsMissingURLAndCancelledPing(t *testing.T) {
	if db, err := Connect(context.Background(), ""); err == nil || db != nil || !strings.Contains(err.Error(), "url is not set") {
		t.Fatalf("Connect(empty URL) = (%v, %v), want configuration error", db, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db, err := Connect(ctx, "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	if err == nil || db != nil {
		if db != nil {
			_ = db.Close()
		}
		t.Fatalf("Connect(cancelled context) = (%v, %v), want failed ping", db, err)
	}
}

func TestMigrateRejectsMissingURL(t *testing.T) {
	if err := Migrate(nil, ""); err == nil || !strings.Contains(err.Error(), "URL is not set") {
		t.Fatalf("Migrate(empty URL) error = %v, want configuration error", err)
	}
}
