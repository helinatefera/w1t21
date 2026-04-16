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
docker-compose up
```

This single command builds and starts all three services (PostgreSQL, backend API, frontend). The backend automatically runs database migrations and seeds initial data on first startup.

## Service URLs and Ports

| Service | URL | Port |
|---------|-----|------|
| Frontend (UI) | http://localhost | 80 |
| Backend (API) | http://localhost:8080 | 8080 |
| PostgreSQL | localhost | 5432 |

The frontend Nginx proxy forwards `/api/*` requests to the backend, so all API calls work through http://localhost/api/ as well.

## Demo Credentials

The database is automatically seeded on first startup. Use these credentials to log in:

| Username | Password | Role | Description |
|----------|----------|------|-------------|
| `admin` | `testpass123` | administrator | Full admin access: user management, moderation, analytics, A/B tests, IP rules |
| `seller1` | `testpass123` | seller | Can create/manage collectible listings, confirm/process/complete orders |
| `buyer1` | `testpass123` | buyer | Can browse catalog, place orders, send messages, post reviews |
| `analyst1` | `testpass123` | compliance_analyst | Read-only access to analytics, A/B test results, anomaly alerts |

> **Security:** These are development-only credentials with intentionally weak passwords. Never use them in production. For production deployment, use `POST /api/setup/admin` to bootstrap a secure administrator account.

## Verification

1. Run `docker-compose up` and wait for all services to start
2. Open http://localhost in a browser
3. Log in as `admin` / `testpass123`
4. You should see the Dashboard with stats cards (Open Orders, Unread Notifications, Roles)
5. Navigate to **Catalog** - three sample collectibles appear
6. Navigate to **Users** (admin sidebar) to manage users and roles
7. Navigate to **Analytics** to see funnel, retention, and content performance charts

## API Endpoints

### Setup
- `GET /api/setup/status` - Check whether initial setup is complete (public)
- `POST /api/setup/admin` - Bootstrap the first administrator account (public, disabled after first admin created)

### Authentication
- `POST /api/auth/login` - Login (public, rate limited)
- `POST /api/auth/refresh` - Refresh tokens (public)
- `GET /api/auth/me` - Get current user profile (authenticated)
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
- `POST /api/collectibles/:id/reviews` - Post a review (authenticated)
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
- `POST /api/orders/:id/refund` - Approve refund (seller)
- `POST /api/orders/:id/arbitration` - Open arbitration (buyer or seller)
- `PATCH /api/orders/:id/fulfillment` - Update fulfillment tracking (seller)

### Messages
- `GET /api/orders/:orderId/messages` - List messages (paginated)
- `POST /api/orders/:orderId/messages` - Send message with optional attachment (multipart, 10MB limit)
- `GET /api/messages/:messageId/attachment` - Download attachment

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

### A/B Tests (administrator, compliance_analyst)
- `POST /api/ab-tests` - Create test
- `GET /api/ab-tests` - List tests
- `GET /api/ab-tests/:id` - Get test with results
- `PATCH /api/ab-tests/:id` - Update test
- `POST /api/ab-tests/:id/complete` - Complete test
- `POST /api/ab-tests/:id/rollback` - Rollback test
- `GET /api/ab-tests/assignments` - Get user's variant assignments (all authenticated)
- `GET /api/ab-tests/registry` - Get experiment registry (all authenticated)

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

This runs unit tests (Go backend, Vitest frontend), and API integration tests against a Dockerized environment. No manual setup is required.

- `unit_tests/` - Frontend logic tests (state transitions, validation, PII detection, store behavior)
- `frontend/src/**/*.test.{ts,tsx}` - Frontend component, store, and page tests
- `backend/internal/**/*_test.go` - Backend Go unit tests
- `API_tests/` - Functional API tests (auth, CRUD, permissions, error handling)
- `e2e/` - Browser-level end-to-end tests (Playwright) for critical user journeys

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| DB_PASSWORD | changeme | PostgreSQL password |
| JWT_SIGNING_KEY | dev_jwt_secret_change_in_production | JWT signing key (development only) |
| AES_MASTER_KEY | (dev key) | 256-bit hex key for AES encryption (development only) |
| LISTEN_ADDR | :8080 | Backend listen address |
| DATABASE_URL | (auto-composed) | PostgreSQL connection string |

## Project Structure

```
.
├── docker-compose.yml       # Service orchestration
├── README.md                # This file
├── run_tests.sh             # Test runner script
├── unit_tests/              # Frontend unit tests
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
    ├── seed.sql             # Development sample data
    └── generate-keys.sh     # Key generation utility
```
