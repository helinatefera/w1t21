-- Seed data for LedgerMint development environment
-- WARNING: These are DEVELOPMENT-ONLY credentials. Never use in production.
-- All seed passwords are intentionally weak and publicly known.

-- Create a bootstrap admin for development seeding purposes.
-- In production, use POST /api/setup/admin instead.
-- bcrypt hash of 'testpass123' at cost 12
INSERT INTO users (id, username, password_hash, display_name) VALUES
  ('00000000-0000-0000-0000-000000000001', 'admin', '$2a$12$6AhMnKSqyDHEGsTlLPTuPuGD4xvnWtU0qnbTT3hM/6fonADlvDf5.', 'System Administrator')
ON CONFLICT (username) DO NOTHING;

INSERT INTO user_roles (user_id, role_id, granted_by)
SELECT '00000000-0000-0000-0000-000000000001', id, '00000000-0000-0000-0000-000000000001'
FROM roles WHERE name = 'administrator'
ON CONFLICT DO NOTHING;

-- Create sample users (password for all: testpass123)
-- bcrypt hash of 'testpass123' at cost 12
INSERT INTO users (id, username, password_hash, display_name, created_by) VALUES
  ('00000000-0000-0000-0000-000000000002', 'seller1', '$2a$12$6AhMnKSqyDHEGsTlLPTuPuGD4xvnWtU0qnbTT3hM/6fonADlvDf5.', 'Alice Seller', '00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000003', 'buyer1', '$2a$12$6AhMnKSqyDHEGsTlLPTuPuGD4xvnWtU0qnbTT3hM/6fonADlvDf5.', 'Bob Buyer', '00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000004', 'analyst1', '$2a$12$6AhMnKSqyDHEGsTlLPTuPuGD4xvnWtU0qnbTT3hM/6fonADlvDf5.', 'Carol Analyst', '00000000-0000-0000-0000-000000000001')
ON CONFLICT (username) DO NOTHING;

-- Assign roles
INSERT INTO user_roles (user_id, role_id, granted_by)
SELECT '00000000-0000-0000-0000-000000000002', id, '00000000-0000-0000-0000-000000000001'
FROM roles WHERE name = 'seller'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id, granted_by)
SELECT '00000000-0000-0000-0000-000000000003', id, '00000000-0000-0000-0000-000000000001'
FROM roles WHERE name = 'buyer'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id, granted_by)
SELECT '00000000-0000-0000-0000-000000000004', id, '00000000-0000-0000-0000-000000000001'
FROM roles WHERE name = 'compliance_analyst'
ON CONFLICT DO NOTHING;

-- Create sample collectibles
INSERT INTO collectibles (id, seller_id, title, description, price_cents, currency, contract_address, chain_id, token_id) VALUES
  ('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'Rare Digital Dragon #001', 'A one-of-a-kind digital dragon collectible with fire breathing animation.', 99900, 'USD', '0x1234567890abcdef1234567890abcdef12345678', 1, '1'),
  ('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002', 'Cyber Punk Portrait #042', 'Limited edition cyberpunk-styled portrait from the Neon Series.', 249900, 'USD', '0x1234567890abcdef1234567890abcdef12345678', 1, '42'),
  ('10000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000002', 'Abstract Waves #007', 'Generative art piece from the Ocean Dreams collection.', 49900, 'USD', '0xabcdefabcdefabcdefabcdefabcdefabcdefabcd', 137, '7')
ON CONFLICT DO NOTHING;
