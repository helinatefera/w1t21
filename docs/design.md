## 1. System Overview

- **Platform**: Digital collectibles exchange running on a closed company LAN (air‑gapped).
- **Primary roles**: Buyer, Seller, Administrator, Compliance Analyst.
- **Core capabilities**:
  - Authentication & role based access control (RBAC)
  - Browse, list, and order limited edition digital collectibles
  - In‑app messaging with attachment scanning (on device pattern detection)
  - Notification center with template subscriptions and manual retry
  - Analytics dashboard (funnels, retention, content performance)
  - A/B testing with deterministic traffic splitting and auto rollback
  - Security: rate limiting, IP allow/deny lists, anomaly detection
  - Audit trails and immutable transaction history
  - Local monitoring via structured logs, metrics, and trace IDs

LedgerMint is designed to run entirely on a closed company LAN with no internet connectivity. The entire stack React frontend, Go backend, and PostgreSQL database is packaged as a Docker Compose application that can be deployed on a single local server. All dependencies (npm packages, Go modules, and base Docker images) are vendored or pre‑loaded so that no external registry access is required at runtime, and the React build is served as static files by the Go backend.

## 2. Design Goals

- Fully functional on an air‑gapped LAN.
- Secure, auditable exchange of digital collectibles.
- Clear separation of concerns: React UI, Go API, PostgreSQL persistence.
- Extensible domain model supporting blockchain fields (contract address, chain ID, token ID, metadata URI).
- Maintainable with unified error codes, idempotency keys, and advisory locks.
- Observable via logs, metrics, and trace IDs stored locally.
- Admin friendly A/B testing and analytics without third party services.

## 3. High Level Architecture

- **Three‑tier deployment**:
  ```
  React SPA (static files)
         │
         ▼
  Go + Echo (REST API)
         │
         ▼
  PostgreSQL (primary data store)
  ```
- **Additional components**:
  - In process cache (hot data, TTL 60s)
  - Background workers (notification retries, analytics rollups)
  - Structured logging and metrics (disk persisted)
  - Idempotency key store (PostgreSQL table)
  - PostgreSQL advisory locks
- **Offline LAN principle**: All components on same LAN; no external calls. React app requires backend.

## 4. Frontend Architecture (React)

- **Framework**: React with TypeScript, React Router, responsive design (desktop/tablet, English UI).
- **Route areas**:
  - `/login` – authentication
  - `/dashboard` – personalized home (owned items, open orders, unread notifications)
  - `/browse` – collectible catalog
  - `/listings` – seller’s listing management
  - `/orders` – order history and tracking
  - `/messages` – in‑app chat threads (per order)
  - `/notifications` – notification center
  - `/admin` – admin dashboard (users, IP lists, anomaly alerts, A/B tests)
  - `/analytics` – internal analytics
- **Major UI components**:
  - Collectible card grid and detail view
  - Order placement modal (idempotency key handling)
  - Chat thread panel (attachment upload, on‑device scanning)
  - Notification center (list, filters, retry button)
  - A/B test configuration panel (start/end date, traffic split, rollback)
  - Analytics charts (funnels, retention)
  - Admin security panel (IP allow/deny lists, anomaly alerts)
- **State management**:
  - React Context / Zustand for global UI state
  - React Query for server state caching
  - Local storage for session token and UI preferences (non‑sensitive)

## 5. Backend Architecture (Go + Echo)

- **API design**: RESTful, versioned (`/api/v1/...`), OpenAPI.
- **Middleware stack**:
  - Authentication (JWT or session cookie)
  - CSRF protection (state changing requests)
  - Rate limiting (per user and per IP)
  - Request validation and injection prevention
  - Structured logging (trace ID, user ID, endpoint, latency)
  - Recovery and unified error codes
- **Service layer (Go packages)**:
  - `AuthService` – login, token refresh, logout, lockout
  - `UserService` – CRUD (admin only), role assignment
  - `CollectibleService` – listing, catalog, inventory
  - `OrderService` – placement (idempotency, advisory locks), status updates
  - `MessageService` – chat threads, attachments, pattern detection
  - `NotificationService` – creation, template subscription, retry
  - `AnalyticsService` – funnel/retention computation, event ingestion
  - `ABTestService` – traffic splitting, variant assignment, auto‑rollback
  - `SecurityService` – IP checks, anomaly detection, alerts
  - `AuditService` – immutable audit logging
- **Background jobs (in‑process)**:
  - Notification retry worker (5 retries, exponential backoff)
  - Analytics rollup worker (hourly/daily)
  - Anomaly detection worker (cancellations, checkout failures)
  - TTL cache cleaner

## 6. Database Design (PostgreSQL)

- **Core tables**:
  - `users` – id, username, password_hash (bcrypt), role, status
  - `collectibles` – id, seller_id, contract_address, chain_id, token_id, metadata_uri, price, quantity_available, status
  - `orders` – id, buyer_id, collectible_id, quantity, total_price, status, idempotency_key
  - `order_events` – id, order_id, event_type, metadata (immutable)
  - `message_threads` – id, order_id, participant_ids
  - `messages` – id, thread_id, sender_id, content, attachment_url, scanned_at
  - `notifications` – id, user_id, template_id, title, body, read, status, retry_count, next_retry_at
  - `notification_templates` – id, name, default_content
  - `user_notification_preferences` – user_id, template_id, enabled
  - `analytics_events` – id, event_type, user_id, collectible_id, session_id, metadata
  - `ab_tests` – id, name, start_date, end_date, control_config, treatment_config, traffic_split, conversion_metric, rollback_threshold, status
  - `ab_test_assignments` – user_id, test_id, variant
  - `ip_allowlist` / `ip_denylist` – id, ip_range
  - `anomaly_alerts` – id, user_id, alert_type, severity, acknowledged
  - `audit_logs` – id, user_id, action, resource_type, old_value, new_value, ip_address, trace_id
- **Constraints**: Unique on `orders(idempotency_key, user_id)`; foreign keys; append‑only for events and audit logs.
- **Encryption at rest**:
  - Sensitive columns (phone, government ID) encrypted with AES‑256.
  - Application managed key stored in encrypted config file, decrypted via env var at startup.
  - Go models mark columns with `@Encrypted`; repository layer handles crypto.

## 7. Security Design

- **Authentication & sessions**:
  - JWT access tokens (15 min) + refresh tokens (7 days, httpOnly cookie).
  - Logout invalidates refresh token server side.
  - “Remember me” extends refresh token lifetime.
- **Password handling**: bcrypt hash (cost 12) – never encrypted. Inline validation feedback.
- **Rate limiting**:
  - Login: 10 attempts/15 min; after 5 failures → 30‑min lockout.
  - Order creation: 30/min per user.
  - Message sending: 20/min per user.
  - Listing creation: 10/hour per seller.
  - IP‑based limits = half of user limits.
- **IP allow/deny lists**: Admin manages ranges; middleware blocks denylist then checks allowlist.
- **Anomaly detection** (configurable):
  - >6 order cancellations in 24h
  - >10 failed checkout attempts in 1h
  - >50 rapid page views (bot‑like)
  - Actions: generate internal alert (admin console), optional auto‑restrict account.
- **CSRF protection**: Double‑submit cookie pattern or SameSite=Strict for state‑changing requests.
- **Input validation & injection prevention**: Go validator library; parameterized queries; no raw SQL concatenation.

## 8. Domain Model Details

- **Collectible**:
  - UUID, seller_id, contract_address, chain_id, token_id, metadata_uri
  - Denormalized: title, description, image_url
  - price (decimal), quantity_available, status (draft/published/sold_out/hidden)
- **Order**:
  - UUID, buyer_id, collectible_id, quantity, total_price
  - status (pending/confirmed/processing/shipped/completed/cancelled/refunded)
  - idempotency_key (client‑generated UUID)
- **Order Event** (immutable): id, order_id, event_type, metadata, created_at
- **Message**: id, thread_id, sender_id, content, attachment_url, scanned_at, created_at
- **Notification**: id, user_id, template_id, title, body, read, status, retry_count, next_retry_at
- **A/B Test**: id, name, start_date, end_date, control_config (JSON), treatment_config (JSON), traffic_split (%), conversion_metric, rollback_threshold (%), status

## 9. Order and Inventory Management

- **Order placement flow**:
  1. Client POST `/orders` with `collectible_id`, `quantity`, `idempotency_key`.
  2. Server checks idempotency key for user – returns existing order if duplicate.
  3. Acquires PostgreSQL advisory lock on `collectible_id` (`pg_advisory_xact_lock`).
  4. Checks available quantity, decrements, creates order with status `pending`.
  5. Releases lock on commit.
  6. Enqueues notifications for buyer and seller.
- **Overselling prevention**: Advisory locks prevent concurrent decrements; timeout (5 sec) → 409 Conflict.
- **Fulfillment tracking**:
  - Seller updates order status (confirmed → processing → shipped/completed).
  - Buyer cancels only if status is `pending` or `confirmed`.
  - Each status change creates an `order_event` and a notification.

## 10. Messaging and On‑Device Pattern Detection

- **Chat threads**: One thread per order (buyer + seller). Sellers respond in‑app.
- **Attachments**:
  - Max size 10 MB; allowed types: images, PDFs.
  - Files stored on server filesystem with random names.
- **On‑device pattern detection (client‑side)**:
  - Regex patterns: US phone numbers (`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`), SSNs (`\b\d{3}-\d{2}-\d{4}\b`).
  - For images: optional OCR (Tesseract.js) extracts text.
  - If match found → message blocked with inline error.
- **Server fallback**: Also scans defensively, but client block is primary.

## 11. Notification Center

- **Notification types** (configurable via templates):
  - Order confirmed, Refund approved, Arbitration opened, Review posted, etc.
- **Template subscriptions**:
  - Users enable/disable templates in notification settings.
  - Preferences stored in `user_notification_preferences`.
- **Retry delivery**:
  - Notifications marked `failed` have a “Retry” button in UI.
  - Manual retry resets retry count and schedules immediate processing.
- **Read/unread state**: Stored as `read` boolean; unread count shown in UI.

## 12. Analytics and A/B Testing

- **Analytics events**:
  - `page_view` (detail, browse), `order_created`, `order_cancelled`, `review_posted`, etc.
  - Stored in `analytics_events` with session ID, user ID, metadata.
- **Funnels & retention**:
  - View‑to‑order conversion = `orders / detail_page_views` per collectible.
  - 7‑day and 30‑day retention = users who ordered and returned.
- **A/B test execution**:
  - Admin creates test with start/end date, traffic split, variants (control/treatment), conversion metric, rollback threshold.
  - Deterministic traffic splitting: `hash(user_id + test_id) % 100 < split` → treatment.
  - Assignment stored in `ab_test_assignments`.
  - Frontend requests variant and renders appropriate UI.
- **Auto‑rollback**:
  - Background job every 5 minutes computes conversion rates (control vs treatment) over last hour.
  - If relative drop `(control - treatment) / control >= threshold`, set test status to `rolled_back`.
  - Frontend polls test status; on rollback, all users see control.
  - Admin receives in app notification.
- **One click rollback**: Button in admin panel to roll back immediately.

## 13. Monitoring and Observability

- **Structured logs**:
  - JSON format, written to disk (`/var/log/ledgermint/app.log`).
  - Fields: timestamp, level, trace_id, user_id, endpoint, latency, error_code.
  - Log rotation using lumberjack (size/time based).
- **Metrics**:
  - Exposed on separate port (`:9090/metrics`) in Prometheus format.
  - Custom metrics: request rate, error rate, order rate, notification queue length.
  - Also written to a time‑series file for local analysis.
- **Trace IDs**:
  - Generated per request, propagated via `X-Trace-Id` header.
  - Included in logs and audit logs.
- **Operator review**:
  - No external monitoring stack required.
  - Operators can tail log files or use a basic admin log viewer (access‑controlled).

## 14. Error Handling and Idempotency

- **Unified error codes**:
  - HTTP status + application error code (e.g., `ERR_ORDER_DUPLICATE`, `ERR_INSUFFICIENT_INVENTORY`).
  - JSON response: `{"error": {"code": "...", "message": "..."}}`
- **Idempotency keys**:
  - Client generates UUID v4 for each order creation.
  - Server stores `(user_id, idempotency_key)` with 24‑hour expiration.
  - Duplicate key → return existing order (idempotent).
- **Idempotent operations**: Order status updates, notification retries.

## 15. Caching Strategy

- **Hot‑data in‑process cache**:
  - Cache collectible catalog tiles (list view) for 60 seconds TTL.
  - Implemented as Go `sync.Map` with timestamps.
  - On update of a collectible, delete cache entry immediately.
- **No external cache (Redis)**: Not required for LAN deployment.

## 16. Background Jobs Design

- **Scheduler**: Go ticker inside main process.
  - Notification retries: every 30 seconds.
  - Analytics rollup: every hour.
  - Anomaly detection: every 15 minutes.
- **Concurrency control**:
  - Use PostgreSQL `SELECT ... FOR UPDATE SKIP LOCKED` to claim jobs.
  - Each job runs in its own goroutine with panic recovery.
- **Idempotent job processing**:
  - Notification retries: update status to `processing` → `sent` or `failed`.
  - Analytics rollups: use date ranges and `ON CONFLICT DO NOTHING`.

## 17. Deployment on LAN

- **Docker Compose example**:
  ```yaml
  version: '3.8'
  services:
    postgres:
      image: postgres:15
      environment:
        POSTGRES_DB: ledgerdb
        POSTGRES_USER: ledger
        POSTGRES_PASSWORD: securepassword
      volumes:
        - pgdata:/var/lib/postgresql/data
    backend:
      build: ./backend
      ports:
        - "8080:8080"
      depends_on:
        - postgres
      environment:
        DB_URL: postgres://ledger:securepassword@postgres/ledgerdb?sslmode=disable
        ENCRYPTION_KEY_FILE: /run/secrets/db_key
      secrets:
        - db_key
    frontend:
      build: ./frontend
      ports:
        - "80:80"
  secrets:
    db_key:
      file: ./secrets/db_key.txt
  ```
- **Offline readiness**:
  - All npm/Go dependencies vendored or mirrored.
  - Docker images pre‑loaded on server.
  - No external DNS or internet required.

## 18. Testing Strategy

- **Unit tests (Go)**:
  - Repository mocking, service logic (idempotency, advisory locks, rate limiting), anomaly detection rules.
- **Component tests (React)**:
  - Login validation, notification center interactions, A/B test variant rendering.
- **Integration tests**:
  - API endpoints with test database, idempotency key end‑to‑end, order placement and inventory decrement.
- **End‑to‑end scenarios**:
  - Buyer browses, places order, receives notification.
  - Seller lists collectible, responds to message.
  - Admin creates A/B test, monitors conversion, triggers rollback.
  - Anomaly detection generates alert.

## 19. Future Extensibility

- Blockchain payment integration (crypto wallets).
- Real‑time notifications via WebSockets (when LAN permits).
- Horizontal scaling with load balancer (requires shared PostgreSQL and cache).
- Export of analytics to external BI tools (CSV/JSON).

## 20. System Hardening Considerations

- **Backup and restore**:
  - PostgreSQL periodic `pg_dump` to encrypted disk.
  - Admin can trigger backup via API or CLI.
- **Disaster recovery**:
  - Restore from latest dump.
  - Idempotent order keys prevent duplicate after restore.
- **Clock skew**:
  - All times stored in UTC.
  - Background jobs use database `NOW()` to avoid reliance on local clock.
- **Multi‑instance safety**:
  - If horizontally scaled, advisory locks and `SKIP LOCKED` work across instances.
  - In‑process cache becomes per‑instance – acceptable for LAN.
