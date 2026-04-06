-- Audit log: append-only table for security and business-critical actions.
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID REFERENCES users(id),
    action VARCHAR(80) NOT NULL,
    resource_type VARCHAR(40) NOT NULL,
    resource_id UUID,
    details JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id, created_at DESC);

-- Append-only protection: prevent updates and deletes on audit_logs.
CREATE OR REPLACE FUNCTION prevent_audit_log_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only: updates and deletes are not allowed';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_audit_log_no_mutation
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_mutation();

-- Fix idempotency scope: replace global UNIQUE on idempotency_key with
-- per-buyer composite uniqueness.  This lets different buyers reuse the
-- same client-generated key without colliding.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_idempotency_key_key;
DROP INDEX IF EXISTS idx_orders_idempotency;
ALTER TABLE orders ADD CONSTRAINT uq_orders_buyer_idempotency UNIQUE (buyer_id, idempotency_key);
CREATE INDEX idx_orders_idempotency ON orders(buyer_id, idempotency_key);
