-- Rolling-window login abuse tracking.
-- Stores individual failed-login timestamps so the auth service can evaluate
-- thresholds within a configurable time window (default 15 minutes).
CREATE TABLE login_attempts (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        REFERENCES users(id) ON DELETE CASCADE,
    ip_address  TEXT        NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_attempts_user_ts ON login_attempts(user_id, attempted_at);
CREATE INDEX idx_login_attempts_ip_ts   ON login_attempts(ip_address, attempted_at);
