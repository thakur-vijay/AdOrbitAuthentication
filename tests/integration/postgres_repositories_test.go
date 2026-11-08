//go:build integration

package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/adorbit/authentication/internal/auth"
	emailotp "github.com/adorbit/authentication/internal/emailOtp"
	appjwt "github.com/adorbit/authentication/internal/jwt"
)

func TestPostgresCredentialRepositoryPersistsAndEnforcesConstraints(t *testing.T) {
	db := postgresFixture(t)
	repository := auth.NewRepository(db)
	ctx := context.Background()
	userID := mustUUID(t)
	credential := auth.Credential{
		UserID:       userID,
		Email:        "CaseSensitive@example.com",
		PasswordHash: "argon2id-test-hash",
	}
	if err := repository.SaveCredentials(ctx, credential); err != nil {
		t.Fatalf("SaveCredentials() error = %v", err)
	}
	got, err := repository.FetchByEmail(ctx, credential.Email)
	if err != nil {
		t.Fatalf("FetchByEmail() error = %v", err)
	}
	if got == nil || got.UserID != userID || got.Email != credential.Email || got.PasswordHash != credential.PasswordHash || got.Status != auth.Pending || got.IsEmailVerified {
		t.Fatalf("FetchByEmail() = %#v, want persisted pending/unverified credential", got)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatalf("database defaults did not set timestamps: %#v", got)
	}
	if missing, err := repository.FetchByEmail(ctx, "missing@example.com"); err != nil || missing != nil {
		t.Fatalf("FetchByEmail(missing) = (%#v, %v), want (nil, nil)", missing, err)
	}
	if lower, err := repository.FetchByEmail(ctx, "casesensitive@example.com"); err != nil || lower != nil {
		t.Fatalf("email lookup case behavior = (%#v, %v), want case-sensitive miss", lower, err)
	}

	verified, err := repository.UpdateCredentialStatus(ctx, userID, auth.Active)
	if err != nil || verified {
		t.Fatalf("UpdateCredentialStatus() = (%v, %v), want (false, nil)", verified, err)
	}
	if err := repository.UpdateIsEmailVerified(ctx, userID); err != nil {
		t.Fatalf("UpdateIsEmailVerified() error = %v", err)
	}
	verified, err = repository.UpdateCredentialStatus(ctx, userID, auth.Active)
	if err != nil || !verified {
		t.Fatalf("UpdateCredentialStatus(verified) = (%v, %v), want (true, nil)", verified, err)
	}
	if _, err := repository.UpdateCredentialStatus(ctx, mustUUID(t), auth.Active); err != nil {
		t.Fatalf("UpdateCredentialStatus(missing) error = %v, want no error", err)
	}
	if _, err := repository.UpdateCredentialStatus(ctx, userID, "invalid-status"); err == nil {
		t.Fatal("UpdateCredentialStatus() accepted a value rejected by the database CHECK constraint")
	}
	if err := repository.UpdateIsEmailVerified(ctx, mustUUID(t)); err != nil {
		t.Fatalf("UpdateIsEmailVerified(missing) error = %v, want no error", err)
	}

	if err := repository.SaveCredentials(ctx, auth.Credential{UserID: mustUUID(t), Email: credential.Email, PasswordHash: "duplicate"}); err == nil {
		t.Fatal("SaveCredentials() accepted duplicate email despite UNIQUE constraint")
	}
	if err := repository.SaveCredentials(ctx, auth.Credential{UserID: userID, Email: "different@example.com", PasswordHash: "duplicate"}); err == nil {
		t.Fatal("SaveCredentials() accepted duplicate user ID despite PRIMARY KEY constraint")
	}
	if err := repository.SaveCredentials(ctx, auth.Credential{UserID: "not-a-uuid", Email: "bad-id@example.com", PasswordHash: "hash"}); err == nil {
		t.Fatal("SaveCredentials() accepted invalid UUID data")
	}

	closedDB, err := sql.Open("pgx", "postgres://invalid:invalid@127.0.0.1:1/not-used?sslmode=disable")
	if err != nil {
		t.Fatalf("open database handle for failure case: %v", err)
	}
	if err := closedDB.Close(); err != nil {
		t.Fatalf("close database handle: %v", err)
	}
	if _, err := auth.NewRepository(closedDB).FetchByEmail(ctx, credential.Email); err == nil {
		t.Fatal("FetchByEmail() hid a closed-database error")
	}
}

func TestPostgresRefreshTokenRepositoryRotatesAndRevokesAtomically(t *testing.T) {
	db := postgresFixture(t)
	repository := appjwt.NewRepository(db)
	ctx := context.Background()
	userID := mustUUID(t)
	now := time.Now().UTC()
	initial := appjwt.RefreshToken{ID: mustUUID(t), UserID: userID, TokenHash: "old-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := repository.Create(ctx, initial); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	fetched, err := repository.GetByHash(ctx, initial.TokenHash)
	if err != nil || fetched.ID != initial.ID || fetched.UserID != userID || fetched.TokenHash != initial.TokenHash || fetched.RevokedAt != nil {
		t.Fatalf("GetByHash() = (%#v, %v), want stored active token", fetched, err)
	}
	if _, err := repository.GetByHash(ctx, "absent-hash"); !errors.Is(err, appjwt.ErrRefreshTokenNotFound) {
		t.Fatalf("GetByHash(missing) error = %v, want ErrRefreshTokenNotFound", err)
	}

	newToken := appjwt.RefreshToken{ID: mustUUID(t), UserID: userID, TokenHash: "new-hash", ExpiresAt: now.Add(2 * time.Hour), CreatedAt: now.Add(time.Second)}
	if err := repository.Rotate(ctx, initial.TokenHash, newToken); err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if _, err := repository.GetByHash(ctx, initial.TokenHash); !errors.Is(err, appjwt.ErrRefreshTokenNotFound) {
		t.Fatalf("old token lookup after rotation error = %v, want not found", err)
	}
	if err := repository.Rotate(ctx, initial.TokenHash, initial); !errors.Is(err, appjwt.ErrRefreshTokenReused) {
		t.Fatalf("second Rotate() error = %v, want ErrRefreshTokenReused", err)
	}
	if err := repository.RevokeByHash(ctx, newToken.TokenHash); err != nil {
		t.Fatalf("RevokeByHash() error = %v", err)
	}
	fetched, err = repository.GetByHash(ctx, newToken.TokenHash)
	if err != nil || fetched.RevokedAt == nil {
		t.Fatalf("GetByHash(revoked) = (%#v, %v), want revocation timestamp", fetched, err)
	}
	if err := repository.RevokeByHash(ctx, "missing-hash"); !errors.Is(err, appjwt.ErrRefreshTokenNotFound) {
		t.Fatalf("RevokeByHash(missing) error = %v", err)
	}
	if err := repository.RevokeByUserID(ctx, userID); !errors.Is(err, appjwt.ErrRefreshTokenNotFound) {
		t.Fatalf("RevokeByUserID(all revoked) error = %v", err)
	}

	// Create upserts a single active refresh token per user, as the current schema intends.
	replacement := appjwt.RefreshToken{ID: mustUUID(t), UserID: userID, TokenHash: "replacement-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := repository.Create(ctx, replacement); err != nil {
		t.Fatalf("Create() upsert error = %v", err)
	}
	if err := repository.RevokeByUserID(ctx, userID); err != nil {
		t.Fatalf("RevokeByUserID() error = %v", err)
	}
	if _, err := repository.GetByHash(ctx, replacement.TokenHash); err != nil {
		t.Fatalf("GetByHash(revoked by user) error = %v", err)
	}

	expiredUserID := mustUUID(t)
	if err := repository.Create(ctx, appjwt.RefreshToken{ID: mustUUID(t), UserID: expiredUserID, TokenHash: "expired-hash", ExpiresAt: now.Add(-time.Minute), CreatedAt: now}); err != nil {
		t.Fatalf("Create(expired) error = %v", err)
	}
	expiredReplacement := appjwt.RefreshToken{ID: mustUUID(t), UserID: expiredUserID, TokenHash: "never-used", ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := repository.Rotate(ctx, "expired-hash", expiredReplacement); !errors.Is(err, appjwt.ErrRefreshTokenReused) {
		t.Fatalf("Rotate(expired) error = %v, want ErrRefreshTokenReused", err)
	}
}

func TestPostgresConcurrentRefreshRotationHasOneWinner(t *testing.T) {
	db := postgresFixture(t)
	repository := appjwt.NewRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	old := appjwt.RefreshToken{ID: mustUUID(t), UserID: mustUUID(t), TokenHash: "shared-old-token", ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := repository.Create(ctx, old); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			replacement := appjwt.RefreshToken{
				ID:        mustUUID(t),
				UserID:    old.UserID,
				TokenHash: fmt.Sprintf("new-token-%d", index),
				ExpiresAt: now.Add(time.Hour),
				CreatedAt: now.Add(time.Duration(index + 1)),
			}
			results <- repository.Rotate(ctx, old.TokenHash, replacement)
		}(i)
	}
	close(start)
	group.Wait()
	close(results)
	wins := 0
	for err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, appjwt.ErrRefreshTokenReused) {
			t.Fatalf("concurrent Rotate() error = %v, want ErrRefreshTokenReused", err)
		}
	}
	if wins != 1 {
		t.Fatalf("concurrent rotation winners = %d, want exactly one", wins)
	}
}

func TestPostgresOTPRepositoryCooldownAttemptsAndOneTimeVerification(t *testing.T) {
	db := postgresFixture(t)
	repository := emailotp.NewRepository(db)
	ctx := context.Background()
	userID := mustUUID(t)
	insertCredential(t, db, userID, "otp@example.com")
	now := time.Now().UTC()
	if err := repository.Upsert(ctx, emailotp.EmailOTP{UserID: userID, OTPHash: "stored-hmac", ExpiresAt: now.Add(5 * time.Minute)}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	got, err := repository.GetByUserID(ctx, userID)
	if err != nil || got.OTPHash != "stored-hmac" || got.Attempts != 0 || got.MaxAttempts != 5 || got.VerifiedAt != nil {
		t.Fatalf("GetByUserID() = (%#v, %v), want stored unverified OTP", got, err)
	}
	allowed, err := repository.CheckResendCooldown(ctx, userID)
	if err != nil || allowed {
		t.Fatalf("CheckResendCooldown(initial) = (%v, %v), want (false, nil)", allowed, err)
	}
	if _, err := db.Exec(`UPDATE email_otps SET last_sent_at = NOW() - INTERVAL '61 seconds' WHERE user_id = $1`, userID); err != nil {
		t.Fatalf("age resend timestamp: %v", err)
	}
	allowed, err = repository.CheckResendCooldown(ctx, userID)
	if err != nil || !allowed {
		t.Fatalf("CheckResendCooldown(after cooldown) = (%v, %v), want (true, nil)", allowed, err)
	}
	allowed, err = repository.CheckResendCooldown(ctx, userID)
	if err != nil || allowed {
		t.Fatalf("CheckResendCooldown(repeat) = (%v, %v), want (false, nil)", allowed, err)
	}
	for i := 0; i < 5; i++ {
		if err := repository.IncrementAttempts(ctx, userID); err != nil {
			t.Fatalf("IncrementAttempts(%d) error = %v", i+1, err)
		}
	}
	got, err = repository.GetByUserID(ctx, userID)
	if err != nil || got.Attempts != 5 {
		t.Fatalf("attempts after five increments = (%#v, %v)", got, err)
	}
	if err := repository.MarkVerified(ctx, userID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("MarkVerified(max attempts) error = %v, want sql.ErrNoRows", err)
	}

	if err := repository.Upsert(ctx, emailotp.EmailOTP{UserID: userID, OTPHash: "replacement-hmac", ExpiresAt: now.Add(5 * time.Minute)}); err != nil {
		t.Fatalf("Upsert(replacement) error = %v", err)
	}
	if err := repository.IncrementAttempts(ctx, userID); err != nil {
		t.Fatalf("IncrementAttempts(replacement) error = %v", err)
	}
	if err := repository.MarkVerified(ctx, userID); err != nil {
		t.Fatalf("MarkVerified() error = %v", err)
	}
	got, err = repository.GetByUserID(ctx, userID)
	if err != nil || got.VerifiedAt == nil || got.Attempts != 1 {
		t.Fatalf("OTP after verification = (%#v, %v)", got, err)
	}
	if err := repository.MarkVerified(ctx, userID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("MarkVerified(second use) error = %v, want sql.ErrNoRows", err)
	}
	if err := repository.Upsert(ctx, emailotp.EmailOTP{UserID: userID, OTPHash: "expired-hmac", ExpiresAt: now.Add(-time.Minute)}); err != nil {
		t.Fatalf("Upsert(expired) error = %v", err)
	}
	if err := repository.MarkVerified(ctx, userID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("MarkVerified(expired) error = %v, want sql.ErrNoRows", err)
	}
	if _, err := repository.GetByUserID(ctx, mustUUID(t)); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetByUserID(missing) error = %v, want sql.ErrNoRows", err)
	}
	if err := repository.Upsert(ctx, emailotp.EmailOTP{UserID: mustUUID(t), OTPHash: "bad-parent", ExpiresAt: now.Add(time.Minute)}); err == nil {
		t.Fatal("Upsert() accepted an OTP without a credential parent")
	}
}

func TestPostgresMigrationsApplyAndRollback(t *testing.T) {
	db := postgresFixture(t)
	for _, table := range []string{"credentials", "jwt_refresh_tokens", "email_otps"} {
		var found bool
		if err := db.QueryRow(`SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&found); err != nil || !found {
			t.Fatalf("table %s was not created by migrations: found=%v error=%v", table, found, err)
		}
	}
	applyDownMigrations(t, db)
	for _, table := range []string{"credentials", "jwt_refresh_tokens", "email_otps"} {
		assertNoTable(t, db, table)
	}
}

func TestPostgresConstraintAndConcurrentCredentialWrites(t *testing.T) {
	db := postgresFixture(t)
	repository := auth.NewRepository(db)
	ctx := context.Background()
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	sharedEmail := "concurrent@example.com"
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			results <- repository.SaveCredentials(ctx, auth.Credential{UserID: mustUUID(t), Email: sharedEmail, PasswordHash: "hash"})
		}()
	}
	close(start)
	group.Wait()
	close(results)
	successes := 0
	failures := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("concurrent duplicate email writes = %d successes, %d errors; want one each", successes, failures)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM credentials WHERE email = $1`, sharedEmail).Scan(&count); err != nil || count != 1 {
		t.Fatalf("credential rows for concurrent email = %d, error %v; want exactly one", count, err)
	}
}
