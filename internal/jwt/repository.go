package jwt

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenRevoked  = errors.New("refresh token revoked")
	ErrRefreshTokenExpired  = errors.New("refresh token expired")
	ErrRefreshTokenReused   = errors.New("refresh token already consumed or invalid")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(
	ctx context.Context,
	token RefreshToken,
) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO jwt_refresh_tokens (
            id,
            user_id,
            token_hash,
            expires_at,
            created_at,
            revoked_at
        )
        VALUES ($1, $2, $3, $4, $5, NULL)

        ON CONFLICT (user_id)
        DO UPDATE SET
            id = EXCLUDED.id,
            token_hash = EXCLUDED.token_hash,
            expires_at = EXCLUDED.expires_at,
            created_at = EXCLUDED.created_at,
            revoked_at = NULL
    `,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
	)

	return err
}

func (r *Repository) GetByHash(
	ctx context.Context,
	tokenHash string,
) (RefreshToken, error) {
	var token RefreshToken
	var revokedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			user_id,
			role,
			token_hash,
			expires_at,
			created_at,
			revoked_at
		FROM jwt_refresh_tokens
		WHERE token_hash = $1
	`,
		tokenHash,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.Role,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&revokedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return RefreshToken{}, ErrRefreshTokenNotFound
	}

	if err != nil {
		return RefreshToken{}, err
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return token, nil
}

func (r *Repository) Rotate(
	ctx context.Context,
	oldTokenHash string,
	newToken RefreshToken,
) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE jwt_refresh_tokens
		SET
			id = $1,
			token_hash = $2,
			expires_at = $3,
			created_at = $4
		WHERE token_hash = $5
		  AND user_id = $6
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`,
		newToken.ID,
		newToken.TokenHash,
		newToken.ExpiresAt,
		newToken.CreatedAt,
		oldTokenHash,
		newToken.UserID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return ErrRefreshTokenReused
	}

	return nil
}

func (r *Repository) RevokeByHash(
	ctx context.Context,
	tokenHash string,
) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE jwt_refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`,
		tokenHash,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}

func (r *Repository) RevokeByUserID(
	ctx context.Context,
	userID string,
) error {
	result, err := r.db.ExecContext(ctx, `
        UPDATE jwt_refresh_tokens
        SET revoked_at = NOW()
        WHERE user_id = $1
          AND revoked_at IS NULL
    `, userID)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}
