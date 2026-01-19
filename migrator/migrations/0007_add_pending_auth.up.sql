CREATE TABLE IF NOT EXISTS pending_oauth_state (
    user_id TEXT NOT NULL,
    state TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, state)
);