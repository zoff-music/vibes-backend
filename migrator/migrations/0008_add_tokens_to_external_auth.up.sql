CREATE TABLE access_tokens (
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    refresh_expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT now,
    updated_at DATETIME DEFAULT now,
    PRIMARY KEY (user_id, provider)
);
