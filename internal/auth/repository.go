package auth

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FetchByEmail(ctx context.Context, email string) (*Credential, error) {
	var credential Credential
	err := r.db.QueryRowContext(ctx, `SELECT user_id, email, password_hash, created_at, updated_at FROM credentials WHERE email = $1`, email).Scan(
		&credential.UserID,
		&credential.Email,
		&credential.PasswordHash,
		&credential.CreatedAt,
		&credential.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &credential, nil
}

func (r *Repository) SaveCredentials(ctx context.Context, credential Credential) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO credentials (user_id, email, password_hash)
		VALUES ($1, $2, $3)
	`,
		credential.UserID,
		credential.Email,
		credential.PasswordHash,
	)

	return err
}
