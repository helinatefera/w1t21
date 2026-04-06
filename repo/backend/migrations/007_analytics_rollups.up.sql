CREATE TABLE analytics_rollups (
    period_days INT PRIMARY KEY,
    views BIGINT NOT NULL DEFAULT 0,
    orders BIGINT NOT NULL DEFAULT 0,
    conversion_rate NUMERIC(8,6) NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE retention_rollups (
    cohort_date DATE PRIMARY KEY,
    cohort_size INT NOT NULL DEFAULT 0,
    retained_count INT NOT NULL DEFAULT 0,
    retention_rate NUMERIC(8,6) NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
