CREATE TABLE auth_tokens (
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    code TEXT NOT NULL,
    state TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, provider)
);
