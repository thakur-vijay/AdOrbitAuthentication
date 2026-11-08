package app

import (
	"testing"

	"github.com/adorbit/authentication/internal/config"
)

func TestNewRejectsNilOrIncompleteConfiguration(t *testing.T) {
	if application, err := New(nil); err == nil || application != nil {
		t.Fatalf("New(nil) = (%v, %v), want configuration error", application, err)
	}
	if application, err := New(&config.Config{}); err == nil || application != nil {
		t.Fatalf("New(empty config) = (%v, %v), want database configuration error", application, err)
	}
}

func TestCloseIsSafeForNilAndEmptyApplications(t *testing.T) {
	var nilApp *App
	if err := nilApp.Close(); err != nil {
		t.Fatalf("Close() on nil App error = %v", err)
	}
	if err := (&App{}).Close(); err != nil {
		t.Fatalf("Close() on empty App error = %v", err)
	}
}
