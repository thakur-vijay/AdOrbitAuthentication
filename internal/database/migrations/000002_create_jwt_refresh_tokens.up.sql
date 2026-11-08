CREATE TABLE
    IF NOT EXISTS jwt_refresh_tokens (
        id UUID PRIMARY KEY,
        user_id UUID NOT NULL,
        token_hash CHAR(64) NOT NULL UNIQUE,
        expires_at TIMESTAMPTZ NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        revoked_at TIMESTAMPTZ NULL
    );

CREATE INDEX idx_jwt_refresh_tokens_user_id ON jwt_refresh_tokens (user_id);

CREATE INDEX idx_jwt_refresh_tokens_expires_at ON jwt_refresh_tokens (expires_at);