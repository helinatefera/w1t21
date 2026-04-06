-- Remove the hardcoded default admin user that was seeded by migration 001.
-- The well-known UUID and username make it a security risk. Administrators
-- must now be created via the bootstrap endpoint (POST /api/setup/admin).
-- Only deletes the user if it still has the original well-known UUID.
DELETE FROM user_roles WHERE user_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000001';
