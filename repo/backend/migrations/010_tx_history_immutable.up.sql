-- Enforce append-only immutability on transaction history
CREATE OR REPLACE FUNCTION prevent_tx_history_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'collectible_tx_history is append-only: updates and deletes are not allowed';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tx_history_no_update ON collectible_tx_history;
CREATE TRIGGER trg_tx_history_no_update
    BEFORE UPDATE OR DELETE ON collectible_tx_history
    FOR EACH ROW EXECUTE FUNCTION prevent_tx_history_mutation();
