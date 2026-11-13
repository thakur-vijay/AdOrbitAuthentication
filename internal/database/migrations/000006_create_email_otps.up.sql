CREATE TABLE
    email_otps (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        user_id UUID NOT NULL REFERENCES credentials (user_id) ON DELETE CASCADE,
        otp_hash VARCHAR(255) NOT NULL,
        expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW () + INTERVAL '5 minutes'),
        attempts SMALLINT NOT NULL DEFAULT 0 CHECK (
            attempts >= 0
            AND attempts <= 5
        ),
        max_attempts SMALLINT NOT NULL DEFAULT 5 CHECK (
            max_attempts > 0
            AND max_attempts <= 5
        ),
        verified_at TIMESTAMPTZ,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        CONSTRAINT email_otps_user_id_unique UNIQUE (user_id)
    );

CREATE INDEX idx_email_otps_expires_at ON email_otps (expires_at);

CREATE INDEX idx_email_otps_unverified ON email_otps (user_id, expires_at)
WHERE
    verified_at IS NULL;
