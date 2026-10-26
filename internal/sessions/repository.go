package sessions

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, session Session) error {
	start := time.Now()

	defer func() {
		log.Printf("[DB] CreateSession: %v", time.Since(start))
	}()
	_, err := r.db.ExecContext(ctx, `INSERT INTO sessions (id, user_id, access_token_hash, refresh_token_hash, expires_at, refresh_expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`, session.ID, session.UserID, session.AccessTokenHash, session.RefreshTokenHash, session.ExpiresAt, session.RefreshExpiresAt, session.CreatedAt)

	return err
}

func (r *Repository) GetByAccessTokenHash(
	ctx context.Context,
	accessTokenHash string,
) (Session, error) {
	var session Session

	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			user_id,
			access_token_hash,
			refresh_token_hash,
			expires_at,
			refresh_expires_at,
			created_at,
			revoked_at
		FROM sessions
		WHERE access_token_hash = $1
	`,
		accessTokenHash,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.AccessTokenHash,
		&session.RefreshTokenHash,
		&session.ExpiresAt,
		&session.RefreshExpiresAt,
		&session.CreatedAt,
		&session.RevokedAt,
	)

	return session, err
}

func (r *Repository) GetByRefreshTokenHash(
	ctx context.Context,
	refreshTokenHash string,
) (Session, error) {
	var session Session

	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			user_id,
			access_token_hash,
			refresh_token_hash,
			expires_at,
			refresh_expires_at,
			created_at,
			revoked_at
		FROM sessions
		WHERE refresh_token_hash = $1
	`,
		refreshTokenHash,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.AccessTokenHash,
		&session.RefreshTokenHash,
		&session.ExpiresAt,
		&session.RefreshExpiresAt,
		&session.CreatedAt,
		&session.RevokedAt,
	)

	return session, err
}

func (r *Repository) UpdateTokens(
	ctx context.Context,
	sessionID string,
	accessTokenHash string,
	refreshTokenHash string,
	expiresAt string,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE sessions
		SET
			access_token_hash = $1,
			refresh_token_hash = $2,
			expires_at = $3
		WHERE id = $4
	`,
		accessTokenHash,
		refreshTokenHash,
		expiresAt,
		sessionID,
	)

	return err
}

func (r *Repository) Revoke(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE id = $1`, sessionID)
	return err
}
