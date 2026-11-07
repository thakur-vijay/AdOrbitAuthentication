package emailotp

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

func (r *Repository) Upsert(ctx context.Context, otp EmailOTP) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO email_otps (
			email,
			otp_hash,
			expires_at,
			attempts,
			verified_at,
			last_sent_at
		)
		VALUES ($1, $2, $3, 0, NULL, NOW())
		ON CONFLICT (email)
		DO UPDATE SET
			otp_hash = EXCLUDED.otp_hash,
			expires_at = EXCLUDED.expires_at,
			attempts = 0,
			verified_at = NULL,
			last_sent_at = NOW(),
			updated_at = NOW()`,
		otp.Email,
		otp.OTPHash,
		otp.ExpiresAt,
	)

	return err
}

func (r *Repository) CheckResendCooldown(
	ctx context.Context,
	userID string,
) (bool, error) {
	var allowed bool

	err := r.db.QueryRowContext(ctx,
		`UPDATE email_otps
		 SET last_sent_at = NOW()
		 WHERE user_id = $1
		   AND last_sent_at <= NOW() - INTERVAL '60 seconds'
		   AND verified_at IS NULL
		 RETURNING TRUE`,
		userID,
	).Scan(&allowed)

	if err == sql.ErrNoRows {
		return false, nil
	}

	return allowed, err
}

func (r *Repository) GetByEmail(
	ctx context.Context,
	email string,
) (*EmailOTP, error) {
	var otp EmailOTP

	err := r.db.QueryRowContext(ctx,
		`SELECT
			email,
			otp_hash,
			expires_at,
			attempts,
			max_attempts,
			verified_at
		FROM email_otps
		WHERE email = $1`,
		email,
	).Scan(
		&otp.Email,
		&otp.OTPHash,
		&otp.ExpiresAt,
		&otp.Attempts,
		&otp.MaxAttempts,
		&otp.VerifiedAt,
	)

	if err != nil {
		return nil, err
	}

	return &otp, nil
}

func (r *Repository) IncrementAttempts(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE email_otps
		 SET attempts = attempts + 1,
		     updated_at = NOW()
		 WHERE email = $1
		   AND attempts < max_attempts
		   AND verified_at IS NULL`,
		email,
	)

	return err
}

func (r *Repository) MarkVerified(ctx context.Context, email string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE email_otps
		 SET verified_at = NOW(),
		     updated_at = NOW()
		 WHERE email = $1
		   AND expires_at > NOW()
		   AND attempts < max_attempts
		   AND verified_at IS NULL`,
		email,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
