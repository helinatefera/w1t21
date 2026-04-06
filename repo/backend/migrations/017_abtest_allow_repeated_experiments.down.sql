DROP INDEX IF EXISTS ab_tests_active_name_unique;

-- Restore the original constraint. This may fail if duplicate names exist
-- across completed/rolled-back rows — manual cleanup would be needed.
ALTER TABLE ab_tests ADD CONSTRAINT ab_tests_name_key UNIQUE (name);
