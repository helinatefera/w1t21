-- Allow multiple A/B tests per experiment name so that successive iterations
-- (e.g. a second "catalog_layout" run after the first was rolled back or
-- completed) can be created without deleting history.
--
-- The old UNIQUE(name) prevented this; replace it with a partial unique
-- index that only enforces uniqueness among *active* tests (draft/running).
-- Completed and rolled-back tests are kept for historical reference.

ALTER TABLE ab_tests DROP CONSTRAINT IF EXISTS ab_tests_name_key;

CREATE UNIQUE INDEX ab_tests_active_name_unique
    ON ab_tests (name)
    WHERE status IN ('draft', 'running');
