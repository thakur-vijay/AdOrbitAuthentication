CREATE TABLE
    credentials (
        user_id UUID PRIMARY KEY,
        password_hash TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE,
        is_email_verified BOOLEAN NOT NULL DEFAULT FALSE,
        status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'failed')),
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW ()
    );