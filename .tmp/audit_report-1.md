# LedgerMint Delivery Acceptance and Project Architecture Audit (Static-Only)

## 1. Verdict
- Overall conclusion: **Partial Pass**

## 2. Scope and Static Verification Boundary
- Reviewed:
  - Documentation/config/startup/test artifacts: `README.md:14`, `README.md:188`, `.env.example:1`, `Makefile:1`, `run_tests.sh:1`.
  - Backend entry, router, middleware, handlers, services, stores, workers, migrations.
  - Frontend routes/pages/api/stores relevant to prompt flows.
  - Test assets (Go tests, shell API tests, JS unit tests) **statically only**.
- Not reviewed:
  - Runtime behavior under real traffic, browser interaction, Docker networking, DB stateful execution.
  - Operational characteristics under concurrency/load in live deployment.
- Intentionally not executed:
  - Project startup, Docker, tests, external services.
- Claims requiring manual verification:
  - End-to-end runtime correctness, UX quality at runtime, actual offline LAN deployment behavior, security effectiveness in production network conditions, and real retry/delivery failure handling paths.

## 3. Repository / Requirement Mapping Summary
- Prompt core goal mapped: offline-capable LAN digital collectibles exchange with buyer/seller/admin/compliance roles, auth, orders, messaging with PII/attachment controls, notification center + retries, analytics + A/B testing + rollback, PostgreSQL SoR, security controls, maintainability primitives.
- Main implementation mapped:
  - Backend APIs and role-gated routes: `backend/internal/router/router.go:36`.
  - Domain/persistence via migrations and stores: `backend/migrations/002_collectibles.up.sql:1`, `backend/migrations/003_orders.up.sql:1`, `backend/migrations/005_notifications.up.sql:1`.
  - Frontend workflows (login/dashboard/catalog/orders/messages/notifications/analytics/admin): `frontend/src/App.tsx:42`.

## 4. Section-by-section Review

### 1. Hard Gates
#### 1.1 Documentation and static verifiability
- Conclusion: **Pass**
- Rationale: Clear startup, setup, endpoint, test, env, and structure docs are present and mostly consistent with code layout.
- Evidence: `README.md:14`, `README.md:73`, `README.md:188`, `README.md:212`, `run_tests.sh:1`, `Makefile:1`.

#### 1.2 Material deviation from Prompt
- Conclusion: **Partial Pass**
- Rationale: Core platform shape exists, but A/B experimentation implementation is inconsistent across backend registry vs frontend support; this weakens prompt-fit for robust admin/compliance experimentation.
- Evidence: backend registry includes `checkout_flow` and `search_ranking` (`backend/internal/service/abtest_registry.go:25`, `backend/internal/service/abtest_registry.go:32`), frontend experiment UI exposes only `catalog_layout` (`frontend/src/pages/ABTestPage.tsx:10`), active UI branching observed only for catalog/item layout (`frontend/src/pages/CatalogPage.tsx:13`, `frontend/src/pages/CollectiblePage.tsx:16`).
- Manual verification note: confirm intended experiment catalog and supported UI branches with product owner.

### 2. Delivery Completeness
#### 2.1 Coverage of explicit core requirements
- Conclusion: **Partial Pass**
- Rationale: Most core requirements are implemented (auth, orders, messaging limits/PII blocking, notifications/prefs/retry action, analytics, A/B deterministic split, rollback path, advisory locks, idempotency, anomaly alerts). Gaps remain around full experiment/component alignment and strict CSRF-all-state-changing interpretation.
- Evidence:
  - Auth/login/refresh/logout and role-gated routes: `backend/internal/router/router.go:47`, `backend/internal/router/router.go:53`.
  - Idempotency/advisory lock/oversell prevention: `backend/internal/service/order_service.go:27`, `backend/internal/service/order_service.go:60`, `backend/internal/service/order_service.go:65`.
  - Messaging 10MB + PII checks: `backend/internal/handler/message_handler.go:62`, `backend/internal/handler/message_handler.go:73`, `backend/cmd/server/main.go:124`.
  - Notifications + retry action + prefs modes: `backend/internal/service/notification_service.go:115`, `backend/internal/service/notification_service.go:126`, `backend/internal/service/notification_service.go:137`, `frontend/src/pages/NotificationsPage.tsx:114`, `frontend/src/pages/NotificationPrefsPage.tsx:22`.
  - Analytics/A-B/rollback: `backend/internal/router/router.go:117`, `backend/internal/service/abtest_service.go:182`, `backend/internal/worker/abtest_evaluator.go:83`, `frontend/src/pages/ABTestPage.tsx:133`.
  - CSRF exemptions: `backend/internal/middleware/csrf.go:29`.

#### 2.2 Basic end-to-end 0→1 deliverable vs partial demo
- Conclusion: **Pass**
- Rationale: Full repo structure, backend+frontend+migrations+tests+docs exist; not a single-file demo.
- Evidence: `README.md:212`, `backend/cmd/server/main.go:45`, `frontend/src/App.tsx:42`, `backend/migrations/001_users_and_roles.up.sql:1`.

### 3. Engineering and Architecture Quality
#### 3.1 Structure and module decomposition
- Conclusion: **Pass**
- Rationale: Clear separation by handlers/services/stores/middleware/workers and frontend page/module structure.
- Evidence: `README.md:225`, `backend/internal/router/router.go:12`, `backend/internal/service/order_service.go:15`, `frontend/src/App.tsx:1`.

#### 3.2 Maintainability and extensibility
- Conclusion: **Partial Pass**
- Rationale: Good foundations (error codes, migrations, workers, cache TTLs, audit tables, AB registry), but experiment registry/front-end branch drift and many non-production JS tests reduce maintainability confidence.
- Evidence: unified error mapping `backend/internal/handler/helpers.go:37`; cache TTLs `backend/internal/service/collectible_service.go:145`; AB registry drift `backend/internal/service/abtest_registry.go:17`, `frontend/src/pages/ABTestPage.tsx:10`; JS placeholder tests `unit_tests/test_validation.test.js:1`, `unit_tests/test_error_codes.test.js:1`.

### 4. Engineering Details and Professionalism
#### 4.1 Error handling/logging/validation/API detail quality
- Conclusion: **Partial Pass**
- Rationale: Strong structured errors and validation patterns, but some strict prompt/security semantics are weakened (CSRF exemptions for state-changing endpoints; cookie `Secure=false` posture).
- Evidence:
  - Unified error envelope and mapping: `backend/internal/handler/helpers.go:27`, `backend/internal/middleware/errorhandler.go:10`.
  - Validation wrapper used per handlers: `backend/internal/handler/helpers.go:17`.
  - Structured logging with request_id: `backend/internal/middleware/requestid.go:19`, `backend/internal/middleware/logging.go:35`.
  - CSRF exemptions: `backend/internal/middleware/csrf.go:29`.
  - Cookies not Secure: `backend/internal/service/auth_service.go:135`, `backend/internal/service/auth_service.go:214`, `backend/internal/middleware/csrf.go:74`.

#### 4.2 Product-like organization vs demo
- Conclusion: **Pass**
- Rationale: Route coverage, role model, DB schema, workers, and admin areas are product-like.
- Evidence: `backend/internal/router/router.go:62`, `backend/migrations/011_audit_and_idempotency.up.sql:1`, `frontend/src/pages/AnomalyAlertsPage.tsx:27`.

### 5. Prompt Understanding and Requirement Fit
#### 5.1 Business goal and constraint fit
- Conclusion: **Partial Pass**
- Rationale: The implementation largely matches the business scenario, but A/B experiment/component alignment and some strict security text interpretations are incomplete.
- Evidence: prompt-aligned components exist across auth/orders/messages/notifications/analytics/admin routes (`backend/internal/router/router.go:47`, `:86`, `:101`, `:108`, `:117`, `:137`), with mismatch in experiment support (`backend/internal/service/abtest_registry.go:25`, `frontend/src/pages/ABTestPage.tsx:10`).
- Manual verification note: confirm whether unsupported experiments are intentionally deferred.

### 6. Aesthetics (frontend/full-stack)
#### 6.1 Visual and interaction quality fit
- Conclusion: **Cannot Confirm Statistically**
- Rationale: Static code shows responsive classes, hierarchy, hover states, and English labels, but visual quality and rendering correctness require runtime/browser validation.
- Evidence: `frontend/src/pages/LoginPage.tsx:66`, `frontend/src/pages/CatalogPage.tsx:39`, `frontend/src/pages/NotificationsPage.tsx:61`, `frontend/src/pages/DashboardPage.tsx:51`.
- Manual verification note: browser walkthrough on desktop/tablet breakpoints required.

## 5. Issues / Suggestions (Severity-Rated)

### Blocker / High
1. **High** - A/B experiment registry and UI implementation are inconsistent
- Conclusion: **Fail**
- Evidence: backend registry defines `catalog_layout`, `checkout_flow`, `search_ranking` (`backend/internal/service/abtest_registry.go:17`), but frontend creation/options only include `catalog_layout` (`frontend/src/pages/ABTestPage.tsx:10`), and UI variant branching is only wired for catalog/detail layout (`frontend/src/pages/CatalogPage.tsx:13`, `frontend/src/pages/CollectiblePage.tsx:16`).
- Impact: Admin/compliance can create tests that have no corresponding frontend behavior, producing misleading analytics and invalid experiment outcomes.
- Minimum actionable fix: make experiment registry single-sourced and enforce parity checks in CI (backend registry + frontend supported branches); either implement missing experiments or remove them from registry.

2. **High** - High-risk security and authorization behavior is under-tested in production Go tests
- Conclusion: **Partial Pass**
- Evidence: test inventory lacks direct Go coverage for object-level auth in order/message/notification services/handlers and broad route authorization matrix (`find backend -name '*_test.go'` output), while many JS tests are placeholders or contract reimplementations rather than backend execution (`unit_tests/test_validation.test.js:1`, `unit_tests/test_new_behaviors.test.js:1`, `unit_tests/test_high_risk_integration.test.js:1`).
- Impact: severe authz/data-isolation defects could remain undetected while tests still pass.
- Minimum actionable fix: add Go httptest and service/store-backed tests for 401/403/object ownership isolation/admin-route enforcement across orders/messages/notifications/anomalies.

### Medium
3. **Medium** - Strict “all state-changing requests enforce CSRF” requirement is not fully met
- Conclusion: **Partial Pass**
- Evidence: CSRF middleware explicitly exempts `POST /api/auth/login` and `POST /api/setup/admin` (`backend/internal/middleware/csrf.go:29`).
- Impact: prompt’s strict wording is not met literally; threat acceptance exists but should be explicitly documented as a policy exception.
- Minimum actionable fix: document formal exception policy and add compensating controls/tests for exempt endpoints; optionally require Origin/Referer checks for setup endpoint.

4. **Medium** - Notification failure/retry path is weakly grounded in real delivery failure conditions
- Conclusion: **Partial Pass**
- Evidence: default delivery hook `nil` marks pending as delivered in LAN mode (`backend/internal/worker/notification_retry.go:14`, `:44`), so failed/permanent-failed paths are not naturally exercised unless synthetic failures exist.
- Impact: retry UX exists but real-world failure behavior may be unrepresentative.
- Minimum actionable fix: define explicit failure conditions in in-app delivery pipeline and add tests covering failed→retry→delivered/permanently_failed lifecycle.

5. **Medium** - Test artifact duplication/drift risk in JS contract tests
- Conclusion: **Partial Pass**
- Evidence: multiple JS tests re-encode backend rules/constants rather than invoking backend code (`unit_tests/test_order_state_machine.test.js:5`, `unit_tests/test_rolling_window_lockout.test.js:5`, `unit_tests/test_abtest_registry.test.js:5`).
- Impact: silent divergence between JS contract files and Go production logic can create false confidence.
- Minimum actionable fix: trim duplicate JS contract tests or generate them from shared source-of-truth; prioritize backend Go tests.

### Low
6. **Low** - “Rich metadata previews” are basic/static rather than clearly rich preview workflows
- Conclusion: **Partial Pass**
- Evidence: listing form captures metadata fields (`frontend/src/pages/CollectibleFormPage.tsx:132`) and detail page displays them (`frontend/src/pages/CollectiblePage.tsx:79`), but no richer metadata preview rendering (e.g., fetched metadata visualization) is present.
- Impact: weaker UX than prompt intent.
- Minimum actionable fix: add metadata URI fetch/preview panel with validation and fallback states.

## 6. Security Review Summary
- Authentication entry points: **Pass**
  - Evidence: login/refresh/logout and JWT middleware flow (`backend/internal/router/router.go:47`, `:53`; `backend/internal/middleware/auth.go:18`; `backend/internal/service/auth_service.go:36`).
- Route-level authorization: **Pass**
  - Evidence: role guards on admin/users/analytics/ab-tests/admin-anomalies (`backend/internal/router/router.go:63`, `:118`, `:126`, `:145`).
- Object-level authorization: **Partial Pass**
  - Evidence: enforced in orders/messages/notifications services (`backend/internal/service/order_service.go:121`, `:140`; `backend/internal/service/message_service.go:40`, `:57`; `backend/internal/service/notification_service.go:101`, `:123`).
  - Reasoning: implementation exists, but static test coverage for these boundaries is limited.
- Function-level authorization: **Pass**
  - Evidence: route wiring + per-action checks in services (`backend/internal/router/router.go:93`, `:97`; `backend/internal/service/order_service.go:211`, `:241`).
- Tenant/user data isolation: **Partial Pass**
  - Evidence: user-scoped listing and checks (`backend/internal/handler/order_handler.go:81`, `backend/internal/service/message_service.go:93`, `backend/internal/service/notification_service.go:89`).
  - Reasoning: logic appears correct; runtime/data-level adversarial verification not performed.
- Admin/internal/debug protection: **Pass**
  - Evidence: admin groups protected by roles (`backend/internal/router/router.go:137`, `:145`). No open debug endpoints found in reviewed scope.

## 7. Tests and Logging Review
- Unit tests: **Partial Pass**
  - Go unit tests exist for validation, error mapping, CSRF middleware behavior, PII extraction/detection, state-machine basics (`backend/internal/dto/validation_test.go:16`, `backend/internal/handler/handler_integration_test.go:17`, `backend/internal/middleware/csrf_test.go:11`, `backend/internal/service/text_extract_test.go:10`).
  - Many JS tests are placeholders or duplicated contracts (`unit_tests/test_validation.test.js:1`, `unit_tests/test_new_behaviors.test.js:1`).
- API/integration tests: **Partial Pass**
  - Shell API tests exist and appear broad (`API_tests/test_auth.sh:1`, `API_tests/test_security.sh:1`, `API_tests/test_high_risk_integration.sh:1`), but were not executed (static-only boundary).
- Logging categories/observability: **Pass**
  - Structured request logs + request IDs + rotating files + metrics logs to disk exist (`backend/internal/middleware/logging.go:35`, `backend/internal/middleware/requestid.go:19`, `backend/cmd/server/main.go:180`, `backend/internal/worker/metrics_writer.go:19`).
- Sensitive-data leakage risk in logs/responses: **Partial Pass**
  - Positive: unified error responses avoid raw dumps (`backend/internal/middleware/errorhandler.go:10`); sensitive field redaction utility exists (`backend/internal/middleware/logging.go:66`).
  - Gap: cannot fully prove “secrets never appear in logs” statically across all call sites; manual log review needed.

## 8. Test Coverage Assessment (Static Audit)

### 8.1 Test Overview
- Unit tests exist:
  - Go: `backend/internal/**/*_test.go` (e.g., `backend/internal/middleware/csrf_test.go:11`).
  - JS/Vitest: `unit_tests/*.test.js` (e.g., `unit_tests/test_order_state_machine.test.js:1`).
- API/integration tests exist: shell scripts in `API_tests/` (e.g., `API_tests/test_auth.sh:1`).
- Frameworks/entry points:
  - Go `go test ./...` via `run_tests.sh` (`run_tests.sh:55`).
  - Vitest via frontend (`run_tests.sh:97`).
  - API shell tests via curl (`run_tests.sh:173`).
- Docs include test command and prerequisites (`README.md:188`, `README.md:194`).

### 8.2 Coverage Mapping Table
| Requirement / Risk Point | Mapped Test Case(s) | Key Assertion / Fixture / Mock | Coverage Assessment | Gap | Minimum Test Addition |
|---|---|---|---|---|---|
| CSRF protection incl. exempt endpoints | `backend/internal/middleware/csrf_test.go:26`, `:41`, `:56`, `:77`, `:97` | Asserts exempt and enforced routes with 403/allow semantics | basically covered | No full route matrix with auth middleware chain | Add httptest integration per state-changing route groups |
| Input validation constraints | `backend/internal/dto/validation_test.go:16`, `:90`, `:196`, `:252` | Validator on production DTOs, min/max/oneof/cidrv4/6 patterns | sufficient | Handler-specific edge cases not all covered | Add handler request/response tests for malformed JSON + boundary errors |
| Error-code strategy | `backend/internal/dto/error_codes_test.go:11`, `backend/internal/handler/handler_integration_test.go:17` | Sentinel->HTTP/code mapping and JSON envelope checks | sufficient | None major | Keep synced with new codes via table-driven tests |
| Order state transition invariants | `backend/internal/model/order_test.go:20` | Valid/invalid transitions, terminal states | basically covered | Service-layer authorization + persistence side effects not covered by Go unit | Add `OrderService.TransitionStatus` tests with mocked store |
| Message PII and attachment filtering | `backend/internal/service/pii_test.go:5`, `backend/internal/service/text_extract_test.go:46`, `backend/internal/handler/attachment_filter_test.go:30` | Regex detection + extraction pipeline + MIME/extension allowlist | basically covered | No handler end-to-end multipart + authz tests in Go | Add httptest multipart send/download authz tests |
| Notification subscription mode behavior | `backend/internal/service/notification_service_test.go:5`, `:31` | status_only vs all_events slug behavior | insufficient | Does not test `NotificationService.Send/List/Retry` with store-backed state transitions | Add service/store tests for pending/failed/permanent/retry manual path |
| A/B deterministic assignment and experiment validation | `backend/internal/service/abtest_registry_test.go:5`, `:66` | ValidateExperiment + deterministic AssignVariant tests | basically covered | Missing test ensuring frontend registry parity with backend registry | Add CI parity test on experiment names/variants |
| Authentication lockout arithmetic | `unit_tests/test_rolling_window_lockout.test.js:5`, `API_tests/test_auth.sh:180` | JS contract math + shell runtime scenario | insufficient | No direct Go unit tests for `AuthService.Login` lockout behavior | Add Go tests with mocked UserStore for thresholds and unlock expiry |
| Object-level authorization (orders/messages/notifications) | (No direct Go tests found for service object checks) | N/A | missing | High-risk data isolation may regress undetected | Add Go service tests for owner/non-owner across read/write actions |
| Admin/internal endpoint protection | shell scripts (e.g., `API_tests/test_security.sh`) not executed | N/A in static run | cannot confirm | Runtime-only checks not statically proven | Add Go router/httptest authz tests independent of Docker |

### 8.3 Security Coverage Audit
- Authentication: **Partial Pass**
  - Middleware and auth flows exist; CSRF middleware has unit tests. Missing direct Go unit tests for lockout/token rotation end-to-end service behavior.
- Route authorization: **Cannot Confirm Statistically**
  - Route guards are defined, but without execution/httptest matrix severe miswiring could remain.
- Object-level authorization: **Fail (coverage)**
  - Production code has checks, but no adequate Go tests prove enforcement across critical object ownership paths.
- Tenant/data isolation: **Fail (coverage)**
  - Similar gap: sparse direct tests for cross-user isolation on orders/messages/notifications.
- Admin/internal protection: **Cannot Confirm Statistically**
  - Guards exist in router; static-only review cannot prove all handlers enforce expected behavior under real auth contexts.

### 8.4 Final Coverage Judgment
- **Fail**
- Boundary explanation:
  - Covered reasonably: DTO validation, CSRF middleware logic, error-code mapping, some state-machine/PII logic.
  - Uncovered high-risk areas: object-level authorization/data isolation and robust authz matrices at backend execution level. Current tests could pass while severe authorization defects remain.

## 9. Final Notes
- This audit is evidence-based and static-only; runtime claims are intentionally avoided.
- Most architectural primitives exist and map to the prompt, but acceptance confidence is reduced by experiment-implementation drift and insufficient high-risk authorization coverage.
