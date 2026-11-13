ALTER TABLE jwt_refresh_tokens
ADD CONSTRAINT uq_jwt_refresh_tokens_user_id UNIQUE (user_id);
