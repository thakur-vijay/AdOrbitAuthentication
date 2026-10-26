CREATE TABLE
    sessions (
        id UUID PRIMARY KEY,
        user_id UUID NOT NULL,
        access_token_hash TEXT NOT NULL UNIQUE,
        refresh_token_hash TEXT NOT NULL UNIQUE,
        expires_at TIMESTAMPTZ NOT NULL,
        refresh_expires_at TIMESTAMPTZ NOT NULL,
        created_at TIMESTAMPTZ NOT NULL,
        revoked_at TIMESTAMPTZ
    );

CREATE INDEX idx_sessions_user_id ON sessions (user_id);

CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);