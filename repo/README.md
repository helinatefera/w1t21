# LedgerMint - Digital Collectibles Exchange

A full-stack platform for listing, buying, and managing digital collectibles (NFTs). Features role-based access control, order state machine with idempotency, real-time notifications, analytics dashboards, A/B testing, and anomaly detection.

## Architecture

| Layer | Technology |
|-------|-----------|
| Frontend | React 18, TypeScript, Tailwind CSS, Vite, React Query, Zustand, Recharts |
| Backend | Go 1.22, Echo v4, PostgreSQL 16, JWT auth, AES-256-GCM encryption |
| Database | PostgreSQL 16 with advisory locks, CIDR-based IP filtering |
| Infrastructure | Docker Compose, Nginx reverse proxy |

## Quick Start

```bash
docker compose up
```

This single command builds and starts all three services (PostgreSQL, backend API, frontend).

## Service URLs and Ports

| Service | URL | Port |
|---------|-----|------|
| Frontend (UI) | http://localhost | 80 |
| Backend (API) | http://localhost:8080 | 8080 |
| PostgreSQL | localhost | 5432 |

The frontend Nginx proxy forwards `/api/*` requests to the backend, so all API calls work through http://localhost/api/ as well.

## First-Time Setup

No default admin credentials are included in the database migrations. On first deployment, you must bootstrap an administrator account:

```bash
# 1. Start services
docker compose up

# 2. Create the initial administrator (interactive)
make create-admin

# Or use the API directly:
curl -X POST http://localhost:8080/api/setup/admin \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<STRONG-PASSWORD>","display_name":"System Administrator"}'
```

The bootstrap endpoint (`POST /api/setup/admin`) is only available when no administrator account exists. It is intentionally CSRF-exempt because it is called before any user session exists and is a one-time bootstrap operation — once an admin account is created, the endpoint returns `409 Conflict` and cannot be used again. All other API endpoints return `503 Service Unavailable` until setup is complete. You can check setup status at any time via `GET /api/setup/status`.

> **Security:** Never use predictable passwords (e.g. `admin123`, `password`) in production. Rotate credentials immediately after deployment. Default credentials must never be committed to version control.

## Development Seed Data

For local development, run `make seed` after startup to populate sample users and collectibles. **These credentials are for development only and must never be used in production.**

```bash
docker compose exec postgres psql -U ledgermint -d ledgermint -f /dev/stdin < scripts/seed.sql
```

The seed script creates an admin and sample users with weak passwords suitable only for local testing.

## Verification

1. Open http://localhost in a browser
2. Complete the initial setup (create an admin account) or run `make seed` for development
3. Log in with the admin credentials you created
4. You should see the Dashboard with stats cards (Open Orders, Unread Notifications, Roles)
5. Navigate to **Catalog** - after seeding, three sample collectibles appear
6. Navigate to **Users** (admin sidebar) to manage users and roles
7. Navigate to **Analytics** to see funnel, retention, and content performance charts

## API Endpoints

### Setup
- `GET /api/setup/status` - Check whether initial setup is complete (public)
- `POST /api/setup/admin` - Bootstrap the first administrator account (public, disabled after first admin created)

### Authentication
- `POST /api/auth/login` - Login (public, rate limited)
- `POST /api/auth/refresh` - Refresh tokens (public)
- `POST /api/auth/logout` - Logout (authenticated)

### Dashboard
- `GET /api/dashboard` - User dashboard stats

### Users (administrator only)
- `POST /api/users` - Create user
- `GET /api/users` - List users (paginated)
- `GET /api/users/:id` - Get user
- `PATCH /api/users/:id` - Update user
- `POST /api/users/:id/roles` - Add role
- `DELETE /api/users/:id/roles/:roleId` - Remove role
- `POST /api/users/:id/unlock` - Unlock account

### Collectibles
- `GET /api/collectibles` - List published collectibles (paginated)
- `GET /api/collectibles/mine` - List seller's own collectibles (seller)
- `GET /api/collectibles/:id` - Get collectible with transaction history
- `POST /api/collectibles` - Create listing (seller, rate limited)
- `PATCH /api/collectibles/:id` - Update listing (seller)
- `PATCH /api/collectibles/:id/hide` - Hide listing (administrator)
- `PATCH /api/collectibles/:id/publish` - Publish listing (administrator)

### Orders
- `POST /api/orders` - Create order (buyer, requires Idempotency-Key header)
- `GET /api/orders` - List orders (paginated, `?role=buyer|seller`)
- `GET /api/orders/:id` - Get order
- `POST /api/orders/:id/confirm` - Confirm order (seller)
- `POST /api/orders/:id/process` - Start processing (seller)
- `POST /api/orders/:id/complete` - Complete order (seller)
- `POST /api/orders/:id/cancel` - Cancel order (buyer or seller)
- `PATCH /api/orders/:id/fulfillment` - Update fulfillment tracking (seller)

### Messages
- `GET /api/orders/:orderId/messages` - List messages (paginated)
- `POST /api/orders/:orderId/messages` - Send message with optional attachment (multipart, 10MB limit)

### Notifications
- `GET /api/notifications` - List notifications (paginated, `?unread=true`)
- `PATCH /api/notifications/:id/read` - Mark read
- `POST /api/notifications/read-all` - Mark all read
- `POST /api/notifications/:id/retry` - Retry failed notification
- `GET /api/notifications/preferences` - Get preferences
- `PUT /api/notifications/preferences` - Update preferences

### Analytics (administrator, compliance_analyst)
- `GET /api/analytics/funnel` - View-to-order funnel (`?days=7|30`)
- `GET /api/analytics/retention` - Retention cohorts (`?days=7|30`)
- `GET /api/analytics/content-performance` - Content performance (`?limit=20`)

### A/B Tests (administrator)
- `POST /api/ab-tests` - Create test
- `GET /api/ab-tests` - List tests
- `GET /api/ab-tests/:id` - Get test with results
- `POST /api/ab-tests/:id/rollback` - Rollback test
- `GET /api/ab-tests/assignments` - Get user's variant assignments (all authenticated)

### Admin (administrator)
- `GET /api/admin/ip-rules` - List IP rules
- `POST /api/admin/ip-rules` - Create IP rule
- `DELETE /api/admin/ip-rules/:id` - Delete IP rule
- `GET /api/admin/anomalies` - List anomaly alerts (administrator, compliance_analyst)
- `PATCH /api/admin/anomalies/:id/acknowledge` - Acknowledge anomaly
- `GET /api/admin/metrics` - System metrics

## Order State Machine

```
pending -> confirmed -> processing -> completed
  |           |
  v           v
cancelled  cancelled
```

Valid transitions:
- `pending` -> `confirmed` or `cancelled`
- `confirmed` -> `processing` or `cancelled`
- `processing` -> `completed`

## Security Features

- **No default credentials** — the first admin must be created via a secure bootstrap endpoint; no passwords are embedded in migrations or committed to version control
- JWT authentication via HttpOnly cookies (15-min access, 7-day refresh with rotation)
- CSRF protection (double-submit cookie pattern)
- AES-256-GCM encryption for PII (emails)
- bcrypt password hashing (cost 12)
- Account lockout after 5 failed logins (30-minute cooldown)
- Per-endpoint rate limiting — user-based **and** IP-based (login, orders, messages, listings)
- CIDR-based IP allow/deny rules
- PII detection in messages (SSN, phone numbers)
- Refresh token family revocation (detects token reuse attacks)
- Append-only audit log for auth events, admin actions, order transitions, and moderation
- Encrypted keyfile secrets required in non-development environments (staging/production)
- Setup guard middleware blocks all API access until initial admin bootstrap is complete

## Background Workers

| Worker | Interval | Purpose |
|--------|----------|---------|
| Notification Retry | 60s | Retries failed notifications with exponential backoff |
| Anomaly Detector | 5min | Detects excessive cancellations and checkout failures |
| A/B Test Evaluator | 5min | Computes conversion rates, auto-rollback if threshold exceeded |
| Metrics Writer | 60s | Writes Prometheus-style metrics to log files |
| Analytics Rollup | 1h | Pre-aggregates funnel and retention metrics |
| Token Cleanup | 1h | Removes expired refresh tokens |

## Running Tests

```bash
./run_tests.sh
```

This runs both unit tests and API tests. API tests require services to be running (`docker compose up`).

- `unit_tests/` - Core logic tests (state transitions, validation, PII detection)
- `API_tests/` - Functional API tests (auth, CRUD, permissions, error handling)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| APP_ENV | (none) | Must be explicitly set to `development` for plaintext env secrets; all other values (including unset) require `SECRETS_KEYFILE` |
| DB_PASSWORD | changeme | PostgreSQL password |
| JWT_SIGNING_KEY | dev_jwt_secret_change_in_production | JWT signing key (development only) |
| AES_MASTER_KEY | (dev key) | 256-bit hex key for AES encryption (development only) |
| LISTEN_ADDR | :8080 | Backend listen address |
| DATABASE_URL | (auto-composed) | PostgreSQL connection string |
| SECRETS_KEYFILE | | Path to AES-256-GCM encrypted keyfile (required when APP_ENV != development) |
| SECRETS_PASSPHRASE | | Passphrase to decrypt the keyfile |

## Project Structure

```
.
├── docker-compose.yml       # Service orchestration
├── README.md                # This file
├── TEST_PROMPT.md           # Test verification document
├── run_tests.sh             # Test runner script
├── unit_tests/              # Unit tests
├── API_tests/               # API integration tests
├── backend/
│   ├── Dockerfile
│   ├── cmd/server/          # Entry point
│   ├── internal/
│   │   ├── cache/           # In-memory cache with TTL
│   │   ├── config/          # Environment configuration
│   │   ├── crypto/          # AES-256-GCM, bcrypt
│   │   ├── dto/             # Request/response types, errors
│   │   ├── handler/         # HTTP handlers
│   │   ├── middleware/      # Auth, CSRF, rate limiting, IP filter, logging
│   │   ├── model/           # Domain models
│   │   ├── router/          # Route definitions
│   │   ├── service/         # Business logic
│   │   ├── store/           # Database access (pgx)
│   │   └── worker/          # Background jobs
│   └── migrations/          # SQL migrations (auto-applied on startup)
├── frontend/
│   ├── Dockerfile
│   ├── nginx.conf           # Reverse proxy config
│   └── src/
│       ├── api/             # API client (axios)
│       ├── components/      # Layout + shared components
│       ├── pages/           # Route pages
│       ├── store/           # Zustand stores
│       ├── types/           # TypeScript types
│       └── utils/           # Formatters, PII detection
└── scripts/
    ├── seed.sql             # Sample data
    └── generate-keys.sh     # Key generation utility
```
