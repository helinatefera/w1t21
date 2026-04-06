DROP TRIGGER IF EXISTS trg_audit_log_no_mutation ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_log_mutation();
DROP TABLE IF EXISTS audit_logs;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS uq_orders_buyer_idempotency;
DROP INDEX IF EXISTS idx_orders_idempotency;
ALTER TABLE orders ADD CONSTRAINT orders_idempotency_key_key UNIQUE (idempotency_key);
CREATE INDEX idx_orders_idempotency ON orders(idempotency_key);
