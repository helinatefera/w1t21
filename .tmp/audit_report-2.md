# LedgerMint Delivery Acceptance + Project Architecture Audit (Static-Only)

## 1. Verdict
- Overall conclusion: **Partial Pass**

## 2. Scope and Static Verification Boundary
- Reviewed:
  - Documentation/config/run artifacts: `README.md`, `.env.example`, `Makefile`, `docker-compose.yml`, `run_tests.sh`
  - Backend architecture: routes, middleware, handlers, services, stores, models, migrations under `backend/internal/**` and `backend/migrations/**`
  - Frontend architecture/UI flows: `frontend/src/**`
  - Test assets: `backend/internal/*test.go`, `unit_tests/*.test.js`, `API_tests/*.sh`
- Not reviewed:
  - Runtime behavior in browser/API server/DB execution (no startup, no HTTP calls executed by this audit)
  - External network behavior, container orchestration behavior, performance under load
- Intentionally not executed:
  - Project startup, Docker, tests, migrations, background workers
- Claims requiring manual verification:
  - Real runtime flow correctness, scheduler job execution timing, observability output files, and API script pass/fail outcomes

## 3. Repository / Requirement Mapping Summary
- Prompt core goal mapped: LAN/offline digital collectibles exchange with buyer/seller/admin/compliance workflows, security controls, analytics + A/B testing, and operational observability.
- Main mapped implementation areas:
  - Auth/security/middleware: `backend/internal/middleware/*`, `backend/internal/service/auth_service.go`, `backend/internal/router/router.go`
  - Domain/persistence: `backend/internal/service/*`, `backend/internal/store/*`, `backend/migrations/*`
  - Frontend flows: login/dashboard/catalog/orders/messages/notifications/admin/analytics pages in `frontend/src/pages/*`
  - Test surface: Go tests + shell API tests + JS tests

## 4. Section-by-section Review

### 1. Hard Gates
#### 1.1 Documentation and static verifiability
- Conclusion: **Pass**
- Rationale: Startup/config/test docs and entry points are present and mostly consistent; project structure is complete and statically traceable.
- Evidence: `README.md:14`, `README.md:188`, `README.md:212`, `.env.example:1`, `docker-compose.yml:1`, `backend/cmd/server/main.go:45`, `frontend/src/App.tsx:32`
- Manual verification note: Runtime startup success is not confirmed statically.

#### 1.2 Material deviation from Prompt
- Conclusion: **Partial Pass**
- Rationale: Core business workflows are implemented, but there are notable deviations in strict prompt constraints (notably system-of-record boundary and validation rigor).
- Evidence: `backend/internal/router/router.go:73`, `backend/internal/router/router.go:86`, `backend/internal/router/router.go:108`, `backend/internal/router/router.go:117`, `backend/internal/router/router.go:124`, `backend/internal/handler/message_handler.go:112`, `backend/internal/store/message_store.go:38`

### 2. Delivery Completeness
#### 2.1 Coverage of explicit core requirements
- Conclusion: **Partial Pass**
- Rationale:
  - Implemented: auth, catalog/orders, order chat, attachment 10MB checks, PII checks, notifications with retry/preferences, analytics, A/B testing, anomaly alerts, admin controls, advisory locks, idempotency, audit logs.
  - Gaps/deviations: message body max-length validation is missing in handler path; attachments persisted on filesystem, not PostgreSQL (prompt asks PostgreSQL as sole system of record for core entities).
- Evidence: `backend/internal/handler/message_handler.go:57`, `backend/internal/dto/request.go:70`, `backend/internal/worker/notification_retry.go:18`, `backend/internal/store/order_store.go:63`, `backend/migrations/011_audit_and_idempotency.up.sql:34`, `backend/internal/handler/message_handler.go:112`, `backend/internal/store/message_store.go:23`

#### 2.2 End-to-end 0→1 deliverable vs partial demo
- Conclusion: **Pass**
- Rationale: Full multi-module backend/frontend, migrations, docs, scripts, and tests exist; not a fragment/demo-only repo.
- Evidence: `README.md:212`, `backend/cmd/server/main.go:45`, `frontend/src/App.tsx:42`, `backend/migrations/001_users_and_roles.up.sql:3`, `API_tests/test_auth.sh:1`

### 3. Engineering and Architecture Quality
#### 3.1 Engineering structure and module decomposition
- Conclusion: **Pass**
- Rationale: Clear layered decomposition (handler/service/store/middleware/model), route grouping by permission, and dedicated workers.
- Evidence: `backend/internal/router/router.go:36`, `backend/internal/service/order_service.go:15`, `backend/internal/store/order_store.go:13`, `backend/internal/worker/scheduler.go:10`

#### 3.2 Maintainability and extensibility
- Conclusion: **Partial Pass**
- Rationale: Good core patterns (error code strategy, migrations, workers, caching), but maintainability risk from tests that mirror/restate logic instead of exercising production code paths.
- Evidence: `backend/internal/dto/errors.go:22`, `backend/internal/handler/helpers.go:37`, `backend/internal/service/authorization_test.go:13`, `unit_tests/test_validation.test.js:10`

### 4. Engineering Details and Professionalism
#### 4.1 Error handling / logging / validation / API design
- Conclusion: **Partial Pass**
- Rationale:
  - Strengths: consistent error envelopes, request IDs, structured logs, role middleware, parameterized SQL usage.
  - Defects: missing strict body-length validation in messaging path; some setup error path uses non-standard error code (`setup_required`) outside unified `ERR_*` strategy.
- Evidence: `backend/internal/middleware/errorhandler.go:10`, `backend/internal/middleware/logging.go:35`, `backend/internal/handler/message_handler.go:57`, `backend/internal/middleware/setup.go:57`, `backend/internal/dto/errors.go:22`

#### 4.2 Product-like vs demo-like shape
- Conclusion: **Pass**
- Rationale: System resembles a product service with auth, admin, analytics, retries, audits, migrations, and role-based UI.
- Evidence: `frontend/src/pages/DashboardPage.tsx:19`, `frontend/src/pages/ABTestPage.tsx:6`, `backend/internal/worker/abtest_evaluator.go:14`, `backend/migrations/010_tx_history_immutable.up.sql:1`

### 5. Prompt Understanding and Requirement Fit
#### 5.1 Business goal / scenario / constraints fit
- Conclusion: **Partial Pass**
- Rationale: Primary user journeys align; however, explicit architectural constraints are not fully met (filesystem attachment storage vs PostgreSQL-only system-of-record wording, validation coverage gap).
- Evidence: `frontend/src/pages/LoginPage.tsx:71`, `frontend/src/pages/DashboardPage.tsx:51`, `frontend/src/pages/MessagesPage.tsx:63`, `backend/internal/handler/message_handler.go:112`, `backend/internal/store/message_store.go:23`

### 6. Aesthetics (frontend)
#### 6.1 Visual/interaction quality fit
- Conclusion: **Pass**
- Rationale: Functional areas are clearly separated; responsive layouts for desktop/tablet are present; hover/disabled states and feedback messages are implemented.
- Evidence: `frontend/src/pages/LoginPage.tsx:66`, `frontend/src/pages/DashboardPage.tsx:51`, `frontend/src/pages/NotificationsPage.tsx:59`, `frontend/src/pages/MessagesPage.tsx:141`, `frontend/src/components/layout/Sidebar.tsx:29`
- Manual verification note: Final visual polish and cross-device rendering still require manual UI run.

## 5. Issues / Suggestions (Severity-Rated)

### Blocker / High
1. **High** - PostgreSQL is not the sole system of record for message attachments
- Conclusion: **Fail**
- Evidence: `backend/cmd/server/main.go:100`, `backend/internal/handler/message_handler.go:112`, `backend/internal/store/message_store.go:23`
- Impact: Core prompt constraint is violated; attachment payload lives in filesystem (`/app/uploads`) with DB only storing path, creating split durability/backup semantics.
- Minimum actionable fix: Store attachment bytes/metadata in PostgreSQL (e.g., bytea/large object table) or revise architecture + prompt alignment explicitly.

2. **High** - Message body length validation path is incomplete
- Conclusion: **Fail**
- Evidence: `backend/internal/handler/message_handler.go:57`, `backend/internal/dto/request.go:70`
- Impact: State-changing endpoint accepts unbounded message text (except non-empty check), violating “every request subject to parameter validation” intent and increasing abuse/storage risk.
- Minimum actionable fix: Validate `body` length server-side (max 5000) in `MessageHandler.Send` or bind through validated DTO.

3. **High** - Significant portions of test suite are placeholder/logic-replication, reducing defect-detection confidence
- Conclusion: **Fail**
- Evidence: `unit_tests/test_error_codes.test.js:1`, `unit_tests/test_validation.test.js:1`, `backend/internal/service/authorization_test.go:13`, `backend/internal/service/auth_lockout_test.go:46`
- Impact: Tests can pass while production-integrated behavior still breaks (authz regressions, lockout edge cases, request wiring errors).
- Minimum actionable fix: Replace mirrored helper tests with production-path tests (handler/service/store with real mocks/fixtures) and remove `expect(true).toBe(true)` stubs.

### Medium
4. **Medium** - Unified error-code strategy is inconsistent in setup guard path
- Conclusion: **Partial Fail**
- Evidence: `backend/internal/middleware/setup.go:57`, `backend/internal/dto/errors.go:22`
- Impact: Clients receive non-standard code (`setup_required`) outside documented `ERR_*` family.
- Minimum actionable fix: Return unified error envelope/code from `dto` constants for setup-required responses.

5. **Medium** - CSRF policy has explicit state-changing exemptions that conflict with strict reading of prompt
- Conclusion: **Partial Fail**
- Evidence: `backend/internal/middleware/csrf.go:29`
- Impact: If prompt interpreted literally (“all state-changing requests”), current exemptions are non-compliant.
- Minimum actionable fix: Either enforce CSRF on exempt endpoints with bootstrap-safe alternative, or document accepted exception in acceptance criteria.

6. **Medium** - A/B test schedule integrity check is missing (`end_date` can be before `start_date`)
- Conclusion: **Fail**
- Evidence: `backend/internal/service/abtest_service.go:30`, `backend/internal/service/abtest_service.go:39`
- Impact: Invalid experiments can be created, undermining analytics validity and rollback logic.
- Minimum actionable fix: Add validation `endDate.After(startDate)` in create/update paths.

### Low
7. **Low** - README permission statement for A/B tests is stale vs code
- Conclusion: **Partial Fail**
- Evidence: `README.md:132`, `backend/internal/router/router.go:126`
- Impact: Operator confusion; docs say “administrator”, code allows administrator + compliance analyst.
- Minimum actionable fix: Update README endpoint permission text.

## 6. Security Review Summary
- Authentication entry points: **Pass**
  - Evidence: login/refresh/logout/me endpoints and JWT cookie auth are implemented with signed token parsing and unauthorized handling (`backend/internal/router/router.go:49`, `backend/internal/middleware/auth.go:21`).
- Route-level authorization: **Pass**
  - Evidence: role-gated route groups for users/admin/analytics/ab-tests (`backend/internal/router/router.go:63`, `backend/internal/router/router.go:118`, `backend/internal/router/router.go:139`).
- Object-level authorization: **Pass**
  - Evidence: order ownership checks, message participant checks, notification ownership checks (`backend/internal/service/order_service.go:121`, `backend/internal/service/message_service.go:57`, `backend/internal/service/notification_service.go:101`).
- Function-level authorization: **Pass**
  - Evidence: transition/refund/arbitration/fulfillment actor checks in service layer (`backend/internal/service/order_service.go:140`, `backend/internal/service/order_service.go:215`, `backend/internal/service/order_service.go:245`, `backend/internal/service/order_service.go:354`).
- Tenant/user data isolation: **Pass** (single-tenant app, per-user isolation present)
  - Evidence: query scoping and ownership checks by authenticated `user_id` (`backend/internal/handler/order_handler.go:62`, `backend/internal/service/notification_service.go:123`).
- Admin/internal/debug protection: **Pass**
  - Evidence: admin routes are under role middleware; no unauthenticated debug routes found (`backend/internal/router/router.go:139`, `backend/internal/router/router.go:146`).

## 7. Tests and Logging Review
- Unit tests: **Partial Pass**
  - Rationale: Go unit tests exist for DTO/middleware/order/auth/PII/text extraction; however many tests replicate logic instead of testing integrated production flows.
  - Evidence: `backend/internal/middleware/csrf_test.go:136`, `backend/internal/service/authorization_test.go:23`
- API/integration tests: **Pass** (existence/intent), **Cannot Confirm Statistically** (runtime outcome)
  - Evidence: `API_tests/test_auth.sh:1`, `API_tests/test_security.sh:1`, `run_tests.sh:126`
- Logging categories/observability: **Partial Pass**
  - Rationale: Structured logs + request IDs + metrics writer to disk are implemented; trace-id semantics are request-id based only.
  - Evidence: `backend/internal/middleware/logging.go:35`, `backend/internal/middleware/requestid.go:24`, `backend/internal/worker/metrics_writer.go:26`
- Sensitive-data leakage risk in logs/responses: **Pass** (static)
  - Rationale: Request body/headers are not logged by default middleware; auth secrets are not explicitly logged.
  - Evidence: `backend/internal/middleware/logging.go:35`, `backend/internal/handler/auth_handler.go:33`

## 8. Test Coverage Assessment (Static Audit)

### 8.1 Test Overview
- Unit tests exist: Go tests under `backend/internal/**` and JS tests under `unit_tests/`.
- API/integration tests exist: shell scripts under `API_tests/`.
- Frameworks: Go `testing`, Vitest, curl-based shell assertions.
- Test entry points documented: `./run_tests.sh`.
- Evidence: `run_tests.sh:47`, `run_tests.sh:88`, `run_tests.sh:126`, `README.md:188`

### 8.2 Coverage Mapping Table
| Requirement / Risk Point | Mapped Test Case(s) | Key Assertion / Fixture / Mock | Coverage Assessment | Gap | Minimum Test Addition |
|---|---|---|---|---|---|
| Auth login + lockout thresholds | `backend/internal/service/auth_lockout_test.go:20`, `API_tests/test_auth.sh:180` | constant checks + 6th-failure lock script | basically covered | Go tests are mostly logic mirroring, not full service with mocked store | Add table-driven service tests with mocked `UserStore` exercising actual `AuthService.Login` branches |
| CSRF enforcement on state-changing endpoints | `backend/internal/middleware/csrf_test.go:136`, `API_tests/test_auth.sh:252` | exhaustive protected route list and token mismatch checks | sufficient | Runtime integration still unconfirmed | Add handler-level integration tests with router + middleware chain for representative protected endpoints |
| Route authorization (role guards) | `backend/internal/handler/authorization_isolation_test.go:217`, `API_tests/test_security.sh:83` | `RequireRole` denies non-role users | basically covered | No end-to-end check for all role-guarded groups | Add API tests for each guarded group (`/users`, `/analytics`, `/admin/*`) by role matrix |
| Object-level authorization (orders/messages/notifications) | `backend/internal/service/authorization_test.go:107`, `API_tests/test_security.sh:84` | outsider denied checks | insufficient | Go tests reimplement decisions instead of calling production service methods | Convert to tests that call real service methods with mocked stores and fixture orders/messages/notifs |
| Idempotency + duplicate order prevention | `API_tests/test_orders.sh` (present), `unit_tests/test_high_risk_integration.test.js:75` | shell checks + JS contract-style logic | insufficient | No Go test exercising `OrderService.Create` with store behavior under duplicate key/concurrency paths | Add Go tests for `OrderService.Create` with mocked store: same buyer/key replay, different buyer/key allowed |
| Oversell/advisory lock invariants | `unit_tests/test_high_risk_integration.test.js:132` | constant/string-level assertions | missing (production-path) | No direct test of `AcquireAdvisoryLock` + `HasActiveOrder` sequencing in service tests | Add Go service test with mock tx/store asserting call order and oversold conflict mapping |
| Message PII + attachment limits | `backend/internal/service/pii_test.go:5`, `backend/internal/service/text_extract_test.go:46`, `API_tests/test_security.sh:141` | regex detection and PDF/image extraction checks | basically covered | Missing strict body length validation test because handler lacks validation | Add handler test for `body` >5000 should return 422 after fix |
| Notification retry and manual retry states | `backend/internal/service/notification_service_test.go:31`, `API_tests/test_notifications.sh` (present) | status-only filtering checks | insufficient | Limited direct tests of retry worker state transitions (`pending/failed/permanently_failed`) | Add worker tests for `NotificationRetryJobWithDelivery` with failing delivery hook and retry backoff expectations |
| A/B deterministic assignment + rollback trigger | `backend/internal/service/abtest_registry_test.go` (exists), `unit_tests/test_high_risk_integration.test.js:28` | deterministic hash assertions | basically covered | Missing validation for invalid date ranges in production create/update path | Add Go tests for `Create`/`Update` rejecting `end <= start` |
| Logging sensitive data leakage | no direct targeted test found | N/A | missing | No test asserts logs exclude secrets/PII fields | Add middleware logger test with fake request containing sensitive headers/body and assert redaction/no body logging |

### 8.3 Security Coverage Audit
- authentication: **Basically covered**
  - Tests exist for JWT parsing/rejection and auth flows, but some are not full service-path tests (`backend/internal/handler/authorization_isolation_test.go:71`, `backend/internal/service/auth_lockout_test.go:147`).
- route authorization: **Basically covered**
  - Middleware tests present; API scripts cover some role checks (`backend/internal/handler/authorization_isolation_test.go:286`, `API_tests/test_security.sh:83`).
- object-level authorization: **Insufficient**
  - Many tests mirror logic via helper functions instead of invoking production services (`backend/internal/service/authorization_test.go:23`). Severe defects could still remain undetected in integration boundaries.
- tenant/data isolation: **Basically covered**
  - Single-tenant scope with user-isolation checks tested in middleware/API scripts.
- admin/internal protection: **Basically covered**
  - Role checks tested and routes are grouped under admin/compliance middleware, but comprehensive endpoint matrix runtime verification is missing.

### 8.4 Final Coverage Judgment
**Partial Pass**
- Major risks covered: CSRF middleware behavior, JWT basics, PII regex/text extraction components, presence of API scripts for auth/orders/security.
- Uncovered/weak areas where severe defects may pass tests: object-level auth integration, oversell/idempotency production-path tests, retry worker lifecycle behavior, and sensitive-log leakage tests.

## 9. Final Notes
- This is a static-only audit; runtime success is not claimed.
- Most core features are present, but high-impact gaps remain in strict requirement alignment and test assurance quality.
