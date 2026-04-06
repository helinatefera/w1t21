DROP TRIGGER IF EXISTS trg_tx_history_no_update ON collectible_tx_history;
DROP FUNCTION IF EXISTS prevent_tx_history_mutation();
