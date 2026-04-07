# LedgerMint Delivery Acceptance + Project Architecture Audit (Static-Only)

## 1. Verdict
- Overall conclusion: **Partial Pass**

## 2. Scope and Static Verification Boundary
- What was reviewed:
  - Documentation/config/test instructions: `README.md`, `.env.example`, `Makefile`, `run_tests.sh`, `docker-compose.yml`
  - Backend entry points, routing, middleware, handlers, services, stores, migrations under `backend/**`
  - Frontend pages/API/store/utils under `frontend/src/**`
  - Test assets under `backend/internal/*test.go`, `unit_tests/*.test.js`, `API_tests/*.sh`
- What was not reviewed:
  - Runtime behavior under real startup, browser rendering at runtime, Docker/network/container behavior, real DB execution results
- What was intentionally not executed:
  - Project startup, Docker, test execution, migrations, background workers
- Claims requiring manual verification:
  - End-to-end runtime correctness, offline-LAN deployment behavior, worker scheduling under load, visual polish across actual devices, and API test pass/fail outcomes

## 3. Repository / Requirement Mapping Summary
- Prompt core goal mapped: offline-capable LAN collectibles exchange with buyer/seller/admin/compliance workflows, auth, order lifecycle, messaging with PII/attachment controls, notification center, analytics, A/B testing, and security controls.
- Mapped implementation areas:
  - Auth/security/middleware: `backend/internal/middleware/*.go`, `backend/internal/service/auth_service.go`, `backend/internal/router/router.go`
  - Core domain/persistence: `backend/internal/service/*.go`, `backend/internal/store/*.go`, `backend/migrations/*.sql`
  - Frontend workflows: `frontend/src/pages/*.tsx`
  - Tests/verification surface: Go tests + shell API scripts + Vitest stubs

## 4. Section-by-section Review

### 1. Hard Gates

#### 1.1 Documentation and static verifiability
- Conclusion: **Pass**
- Rationale: Startup/config/test instructions and entry points are present and statically consistent enough for manual verification setup.
- Evidence: `README.md:14`, `README.md:188`, `README.md:212`, `.env.example:1`, `backend/cmd/server/main.go:45`, `frontend/src/App.tsx:32`
- Manual verification note: Startup success cannot be proven statically.

#### 1.2 Material deviation from Prompt
- Conclusion: **Partial Pass**
- Rationale: Most business flows align, but there are material requirement-fit gaps (strict CSRF interpretation, A/B schedule integrity validation).
- Evidence: `backend/internal/middleware/csrf.go:29`, `backend/internal/service/abtest_service.go:30`, `backend/internal/service/abtest_service.go:39`

### 2. Delivery Completeness

#### 2.1 Coverage of explicit core requirements
- Conclusion: **Partial Pass**
- Rationale: Core flows are broadly implemented (auth, orders, messaging with 10MB limit + PII checks, notifications with retry + preferences, analytics, anomaly alerts, AB tests), but some explicit guardrails are incomplete (A/B end-before-start not rejected; strict “all state-changing CSRF” not met).
- Evidence: `backend/internal/handler/message_handler.go:75`, `backend/internal/service/pii.go:15`, `backend/internal/service/notification_service.go:115`, `backend/internal/worker/anomaly_detector.go:19`, `backend/internal/worker/abtest_evaluator.go:78`, `backend/internal/service/abtest_service.go:30`, `backend/internal/middleware/csrf.go:29`

#### 2.2 End-to-end deliverable vs partial/demo
- Conclusion: **Pass**
- Rationale: Full-stack structure with backend/frontend/migrations/docs/tests exists; not a single-file demo.
- Evidence: `README.md:212`, `backend/cmd/server/main.go:45`, `backend/migrations/001_users_and_roles.up.sql:1`, `frontend/src/App.tsx:42`, `API_tests/test_auth.sh:1`

### 3. Engineering and Architecture Quality

#### 3.1 Structure and module decomposition
- Conclusion: **Pass**
- Rationale: Layered architecture is clear (handler/service/store/middleware/model) with route grouping and worker modules.
- Evidence: `backend/internal/router/router.go:36`, `backend/internal/service/order_service.go:15`, `backend/internal/store/order_store.go:13`, `backend/internal/worker/scheduler.go:10`

#### 3.2 Maintainability and extensibility
- Conclusion: **Partial Pass**
- Rationale: Core architecture is extensible (migrations, standardized DTOs/errors, workers, cache), but test maintainability and defect-detection confidence are weakened by placeholder and logic-mirroring tests.
- Evidence: `backend/internal/dto/errors.go:22`, `unit_tests/test_error_codes.test.js:11`, `unit_tests/test_validation.test.js:9`, `backend/internal/service/authorization_test.go:13`, `run_tests.sh:86`

### 4. Engineering Details and Professionalism

#### 4.1 Error handling, logging, validation, API design
- Conclusion: **Partial Pass**
- Rationale: Strong error envelope and structured logging patterns, but key validation gaps remain (A/B chronological integrity), and strict CSRF requirement is relaxed via exemptions.
- Evidence: `backend/internal/handler/helpers.go:37`, `backend/internal/middleware/logging.go:35`, `backend/internal/service/abtest_service.go:30`, `backend/internal/middleware/csrf.go:29`

#### 4.2 Product-like vs demo-like shape
- Conclusion: **Pass**
- Rationale: Overall implementation resembles a product service with role boundaries, audit logs, anomaly detection, analytics, A/B operations, retries, and migrations.
- Evidence: `backend/internal/store/audit_store.go:14`, `backend/internal/worker/anomaly_detector.go:14`, `backend/internal/worker/notification_retry.go:18`, `frontend/src/pages/AnalyticsPage.tsx:1`, `frontend/src/pages/ABTestPage.tsx:6`

### 5. Prompt Understanding and Requirement Fit

#### 5.1 Business goal/scenario/constraint fit
- Conclusion: **Partial Pass**
- Rationale: Primary scenario fit is good (buyers/sellers/admin/compliance flows implemented), but explicit constraints are partially unmet in strict CSRF semantics and A/B date guardrails.
- Evidence: `frontend/src/pages/LoginPage.tsx:31`, `frontend/src/pages/DashboardPage.tsx:67`, `frontend/src/pages/MessagesPage.tsx:63`, `frontend/src/pages/NotificationsPage.tsx:59`, `backend/internal/middleware/csrf.go:29`, `backend/internal/service/abtest_service.go:30`

### 6. Aesthetics (frontend)

#### 6.1 Visual and interaction quality
- Conclusion: **Pass**
- Rationale: Layout separation, hierarchy, responsive classes, and interaction states are present; English UI and desktop/tablet-oriented grid behavior are statically evident.
- Evidence: `frontend/src/pages/LoginPage.tsx:66`, `frontend/src/pages/DashboardPage.tsx:51`, `frontend/src/pages/ABTestPage.tsx:81`, `frontend/src/pages/MessagesPage.tsx:110`, `frontend/src/components/layout/Sidebar.tsx:29`
- Manual verification note: Final rendering quality and interaction feel still require manual UI run.

## 5. Issues / Suggestions (Severity-Rated)

### High
1. **Severity:** High  
   **Title:** State-changing CSRF policy does not fully match strict prompt requirement  
   **Conclusion:** Fail  
   **Evidence:** `backend/internal/middleware/csrf.go:29`, `backend/internal/router/router.go:49`, `backend/internal/router/router.go:41`  
   **Impact:** Prompt requires CSRF on all state-changing requests; explicit exemptions on `POST /api/auth/login` and `POST /api/setup/admin` break strict compliance interpretation.  
   **Minimum actionable fix:** Enforce CSRF for all state-changing endpoints, or explicitly revise acceptance scope to permit only pre-session bootstrap/login exceptions.

2. **Severity:** High  
   **Title:** A/B test schedule integrity check is missing (`end_date` can be before/equal `start_date`)  
   **Conclusion:** Fail  
   **Evidence:** `backend/internal/service/abtest_service.go:30`, `backend/internal/service/abtest_service.go:34`, `backend/internal/service/abtest_service.go:39`, `backend/internal/service/abtest_service.go:96`  
   **Impact:** Invalid experiments can be created/updated, undermining deterministic experiment windows, rollback expectations, and analytics validity.  
   **Minimum actionable fix:** Reject create/update when `!endDate.After(startDate)` and add test coverage for create/update invalid chronology.

### Medium
3. **Severity:** Medium  
   **Title:** Test assurance is diluted by placeholder and logic-mirroring tests  
   **Conclusion:** Partial Fail  
   **Evidence:** `unit_tests/test_error_codes.test.js:11`, `unit_tests/test_validation.test.js:9`, `backend/internal/service/authorization_test.go:13`, `backend/internal/service/auth_lockout_test.go:46`, `run_tests.sh:86`  
   **Impact:** Test suite can report success while production-path regressions remain undetected (especially authorization integration and real service/store interactions).  
   **Minimum actionable fix:** Replace placeholders and mirrored-logic tests with production-path tests (handler/service/store with mocks/fixtures) for high-risk authz/lockout/idempotency flows.

4. **Severity:** Medium  
   **Title:** README A/B role permissions are inconsistent with implemented route policy  
   **Conclusion:** Partial Fail  
   **Evidence:** `README.md:132`, `backend/internal/router/router.go:124`, `backend/internal/router/router.go:129`  
   **Impact:** Operational misunderstanding of who can create/update/rollback tests.  
   **Minimum actionable fix:** Update README endpoint role notes to match router behavior.

### Low
5. **Severity:** Low  
   **Title:** Analytics query parameter constraints differ from strict 7/30-day wording  
   **Conclusion:** Partial Fail  
   **Evidence:** `README.md:128`, `backend/internal/handler/analytics_handler.go:28`, `backend/internal/handler/analytics_handler.go:41`  
   **Impact:** API accepts broader `days` values than strict “7/30-day retention/funnel” phrasing, which can create reporting inconsistency with business expectations.  
   **Minimum actionable fix:** Enforce `days in {7,30}` or clearly document broader allowed values.

## 6. Security Review Summary

- authentication entry points: **Pass**  
  Evidence: `backend/internal/router/router.go:48`, `backend/internal/handler/auth_handler.go:24`, `backend/internal/middleware/auth.go:18`  
  Reasoning: Login/refresh/logout/me paths and JWT cookie auth are implemented with token verification and unauthorized handling.

- route-level authorization: **Pass**  
  Evidence: `backend/internal/router/router.go:63`, `backend/internal/router/router.go:117`, `backend/internal/router/router.go:139`  
  Reasoning: Role-based groups for admin/compliance/seller/buyer operations are defined at route level.

- object-level authorization: **Pass**  
  Evidence: `backend/internal/service/order_service.go:121`, `backend/internal/service/message_service.go:89`, `backend/internal/service/notification_service.go:101`  
  Reasoning: Services enforce order participant/owner checks before returning or mutating objects.

- function-level authorization: **Pass**  
  Evidence: `backend/internal/service/order_service.go:140`, `backend/internal/service/order_service.go:215`, `backend/internal/service/order_service.go:245`, `backend/internal/service/order_service.go:354`  
  Reasoning: Sensitive state transitions/refund/arbitration/fulfillment actions include function-level actor checks.

- tenant/user isolation: **Pass** (single-tenant app; per-user isolation present)  
  Evidence: `backend/internal/handler/order_handler.go:62`, `backend/internal/service/notification_service.go:123`, `backend/internal/service/message_service.go:136`  
  Reasoning: User-scoped access checks prevent cross-user data exposure in core flows.

- admin/internal/debug protection: **Pass**  
  Evidence: `backend/internal/router/router.go:139`, `backend/internal/router/router.go:146`  
  Reasoning: Admin/internal endpoints are role-gated; no unprotected debug endpoints were found in reviewed scope.

## 7. Tests and Logging Review

- Unit tests: **Partial Pass**  
  Rationale: Many real Go tests exist for DTO/middleware/handlers/services, but some high-risk tests mirror logic or are placeholders, reducing confidence.  
  Evidence: `backend/internal/dto/validation_test.go:16`, `backend/internal/handler/handler_integration_test.go:17`, `backend/internal/service/authorization_test.go:13`, `unit_tests/test_validation.test.js:9`

- API/integration tests: **Pass** (existence), **Cannot Confirm Statistically** (runtime outcome)  
  Evidence: `API_tests/test_auth.sh:1`, `API_tests/test_security.sh:1`, `run_tests.sh:126`

- Logging categories / observability: **Pass**  
  Rationale: Structured request logs + request IDs + file-backed rotating logs + metrics-to-disk worker are present.  
  Evidence: `backend/internal/middleware/logging.go:35`, `backend/internal/middleware/requestid.go:24`, `backend/cmd/server/main.go:179`, `backend/internal/worker/metrics_writer.go:26`

- Sensitive-data leakage risk in logs/responses: **Partial Pass**  
  Rationale: Request body/headers are not logged by structured logger, but no direct test asserts secret/PII non-leakage in logs.  
  Evidence: `backend/internal/middleware/logging.go:35`, `backend/internal/handler/auth_handler.go:33`, `backend/internal/middleware/logging_test.go:1`

## 8. Test Coverage Assessment (Static Audit)

### 8.1 Test Overview
- Unit tests: **Exist** (Go tests and Vitest files)
- API/integration tests: **Exist** (shell scripts)
- Frameworks: Go `testing`, Vitest, curl-based shell assertions
- Test entry points documented: yes (`./run_tests.sh`)
- Documentation provides test commands: yes
- Evidence: `README.md:188`, `run_tests.sh:47`, `run_tests.sh:88`, `run_tests.sh:126`, `frontend/vitest.config.ts:1`

### 8.2 Coverage Mapping Table

| Requirement / Risk Point | Mapped Test Case(s) | Key Assertion / Fixture / Mock | Coverage Assessment | Gap | Minimum Test Addition |
|---|---|---|---|---|---|
| Login/auth basics and JWT handling | `backend/internal/service/auth_lockout_test.go:147`, `backend/internal/handler/authorization_isolation_test.go:71`, `API_tests/test_auth.sh:1` | Token round-trip parsing and unauthorized checks | basically covered | Many checks are unit-level; full router+cookie flow not statically guaranteed | Add handler+router integration tests for login/refresh/logout cookie lifecycle and CSRF header coupling |
| CSRF enforcement on state-changing routes | `backend/internal/middleware/csrf_test.go:136`, `API_tests/test_auth.sh:252` | Middleware rejects missing/mismatched token | basically covered | Exemptions for login/setup are not tested against prompt strictness policy | Add explicit policy tests asserting intended exemption list and security rationale |
| Route authorization (roles) | `backend/internal/handler/authorization_isolation_test.go:286`, `API_tests/test_security.sh:83` | `RequireRole` denies non-role access | basically covered | No exhaustive endpoint-role matrix tests | Add table-driven role matrix tests for `/users`, `/analytics`, `/ab-tests`, `/admin/*` |
| Object-level authorization (order/message/notification ownership) | `backend/internal/handler/authorization_isolation_test.go:217`, `backend/internal/service/authorization_test.go:23` | Outsider-denied behavior | insufficient | `authorization_test.go` mirrors logic rather than invoking production service methods | Add service tests using mocked stores to exercise real `OrderService` / `MessageService` / `NotificationService` methods |
| Idempotency + oversell safeguards | `backend/internal/model/order_oversell_test.go:53`, `API_tests/test_orders.sh:1` | Oversell/idempotency invariants | basically covered | Limited direct service-layer tests around transaction/advisory-lock failure branches | Add `OrderService.Create` tests for lock acquisition failures, duplicate keys, and race-like sequences |
| Message body/attachment/PII enforcement | `backend/internal/handler/message_body_validation_test.go:16`, `backend/internal/service/pii_test.go:5`, `backend/internal/service/text_extract_test.go:46`, `API_tests/test_messages.sh:1` | 10MB/length/PII pattern checks | sufficient | Runtime MIME edge cases still manual | Add cases for ambiguous MIME + extension combinations through handler path |
| Notification retry lifecycle | `backend/internal/worker/notification_retry_test.go:1`, `API_tests/test_notifications.sh:1` | Pending/failed/permanently_failed transitions | basically covered | End-to-end worker scheduling behavior not statically provable | Add integration-like tests with deterministic clock/backoff assertions |
| A/B deterministic split and rollback | `backend/internal/service/abtest_registry_test.go:1`, `unit_tests/test_abtest_registry.test.js:1` | Deterministic hashing/registry invariants | insufficient | No test for create/update date chronology (`end > start`) | Add `ABTestService.Create/Update` tests for invalid start/end ordering |
| Sensitive log exposure | `backend/internal/middleware/logging_test.go:1` | Logger behavior test | insufficient | No explicit assertion that secrets/PII are absent from emitted log entries | Add logging tests with synthetic sensitive payload and assert redaction/non-emission |

### 8.3 Security Coverage Audit
- authentication: **Basically covered**  
  Tests cover JWT and auth branches, but not full production integration chain.
- route authorization: **Basically covered**  
  Role middleware and selected API paths are tested.
- object-level authorization: **Insufficient**  
  Significant reliance on logic-replication tests means severe integration defects could still pass.
- tenant/data isolation: **Basically covered**  
  Ownership/participant checks are present in tests, though mostly not full-stack.
- admin/internal protection: **Basically covered**  
  Route guards are tested in parts, but exhaustive endpoint matrix is missing.

### 8.4 Final Coverage Judgment
**Partial Pass**

- Major risks covered: CSRF middleware behavior, core auth token behaviors, many DTO validations, message PII/attachment rules, presence of API security scripts.
- Major uncovered/weak risks: object-level auth integration confidence, A/B date chronology enforcement tests, explicit sensitive-log leakage assertions, full endpoint role matrix.
- Boundary: current tests could still pass while some severe authorization/validation regressions remain undetected.

## 9. Final Notes
- This report is static-only and evidence-based; no runtime success is claimed.
- Previous findings about filesystem attachment storage and missing message length enforcement are no longer valid in the current code state (PostgreSQL attachment table + 10,000-char server-side check are present).
- Areas marked Cannot Confirm Statistically require manual execution to verify runtime behavior.
