CREATE TABLE auth_tokens (
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    code TEXT NOT NULL,
    state TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT now,
    updated_at DATETIME DEFAULT now,
    PRIMARY KEY (user_id, provider)
);
