# Combined Audit Report: Test Coverage + README

Date: 2026-04-16
Scope: static inspection only
Repository: repo/

---

## 1. Test Coverage Audit

### Project Type Detection
- README declares full-stack at repo/README.md:3.
- Effective type for this audit: fullstack.

### Strict Definitions (Applied)
- Endpoint = unique METHOD + fully resolved PATH from router groups and prefixes under repo/backend/internal/router/router.go:36-148.
- Endpoint counted as covered only when API tests send request to exact METHOD + normalized PATH.
- True no-mock API test classification requires real HTTP request path in API tests plus no static mocking markers.
- Static-only rule respected: no runtime execution assumptions used.

### Backend Endpoint Inventory
Source of truth: repo/backend/internal/router/router.go:40-148 (58 routes).

1. GET /api/setup/status
2. POST /api/setup/admin
3. POST /api/auth/login
4. POST /api/auth/refresh
5. GET /api/auth/me
6. POST /api/auth/logout
7. GET /api/dashboard
8. POST /api/users
9. GET /api/users
10. GET /api/users/:id
11. PATCH /api/users/:id
12. POST /api/users/:id/roles
13. DELETE /api/users/:id/roles/:roleId
14. POST /api/users/:id/unlock
15. GET /api/collectibles
16. GET /api/collectibles/mine
17. GET /api/collectibles/:id
18. POST /api/collectibles
19. PATCH /api/collectibles/:id
20. POST /api/collectibles/:id/reviews
21. PATCH /api/collectibles/:id/hide
22. PATCH /api/collectibles/:id/publish
23. POST /api/orders
24. GET /api/orders
25. GET /api/orders/:id
26. POST /api/orders/:id/confirm
27. POST /api/orders/:id/process
28. POST /api/orders/:id/complete
29. POST /api/orders/:id/cancel
30. POST /api/orders/:id/refund
31. POST /api/orders/:id/arbitration
32. PATCH /api/orders/:id/fulfillment
33. GET /api/orders/:orderId/messages
34. POST /api/orders/:orderId/messages
35. GET /api/messages/:messageId/attachment
36. GET /api/notifications
37. PATCH /api/notifications/:id/read
38. POST /api/notifications/read-all
39. POST /api/notifications/:id/retry
40. GET /api/notifications/preferences
41. PUT /api/notifications/preferences
42. GET /api/analytics/funnel
43. GET /api/analytics/retention
44. GET /api/analytics/content-performance
45. POST /api/ab-tests
46. GET /api/ab-tests
47. GET /api/ab-tests/:id
48. PATCH /api/ab-tests/:id
49. POST /api/ab-tests/:id/complete
50. POST /api/ab-tests/:id/rollback
51. GET /api/ab-tests/assignments
52. GET /api/ab-tests/registry
53. GET /api/admin/ip-rules
54. POST /api/admin/ip-rules
55. DELETE /api/admin/ip-rules/:id
56. GET /api/admin/metrics
57. GET /api/admin/anomalies
58. PATCH /api/admin/anomalies/:id/acknowledge

### API Test Mapping Table
Classification rule used:
- true no-mock HTTP: curl against API route in API_tests with no static mocking markers.

| Endpoint | Covered | Test type | Test files | Evidence |
|---|---|---|---|---|
| GET /api/setup/status | yes | true no-mock HTTP | API_tests/test_user_mgmt.sh | GET call at test_user_mgmt.sh:60 |
| POST /api/setup/admin | yes | true no-mock HTTP | API_tests/test_user_mgmt.sh | POST call at test_user_mgmt.sh:73 |
| POST /api/auth/login | yes | true no-mock HTTP | API_tests/test_auth.sh | POST call at test_auth.sh:43 |
| POST /api/auth/refresh | yes | true no-mock HTTP | API_tests/test_auth.sh | POST call at test_auth.sh:133 |
| GET /api/auth/me | yes | true no-mock HTTP | API_tests/test_user_mgmt.sh | GET call at test_user_mgmt.sh:87 |
| POST /api/auth/logout | yes | true no-mock HTTP | API_tests/test_auth.sh | POST call at test_auth.sh:149 |
| GET /api/dashboard | yes | true no-mock HTTP | API_tests/test_auth.sh | GET call at test_auth.sh:116 |
| POST /api/users | yes | true no-mock HTTP | API_tests/test_auth.sh | POST call at test_auth.sh:199 |
| GET /api/users | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:165 |
| GET /api/users/:id | yes | true no-mock HTTP | API_tests/test_user_mgmt.sh | GET call at test_user_mgmt.sh:109 |
| PATCH /api/users/:id | yes | true no-mock HTTP | API_tests/test_user_mgmt.sh | PATCH call at test_user_mgmt.sh:151 |
| POST /api/users/:id/roles | yes | true no-mock HTTP | API_tests/test_notifications.sh | POST call at test_notifications.sh:192 |
| DELETE /api/users/:id/roles/:roleId | yes | true no-mock HTTP | API_tests/test_user_mgmt.sh | DELETE call at test_user_mgmt.sh:193 |
| POST /api/users/:id/unlock | yes | true no-mock HTTP | API_tests/test_auth.sh | POST call at test_auth.sh:237 |
| GET /api/collectibles | yes | true no-mock HTTP | API_tests/test_collectibles.sh | GET call at test_collectibles.sh:77 |
| GET /api/collectibles/mine | yes | true no-mock HTTP | API_tests/test_collectibles.sh | GET call at test_collectibles.sh:188 |
| GET /api/collectibles/:id | yes | true no-mock HTTP | API_tests/test_collectibles.sh | GET call at test_collectibles.sh:99 |
| POST /api/collectibles | yes | true no-mock HTTP | API_tests/test_collectibles.sh | POST call at test_collectibles.sh:126 |
| PATCH /api/collectibles/:id | yes | true no-mock HTTP | API_tests/test_collectibles.sh | PATCH call at test_collectibles.sh:177 |
| POST /api/collectibles/:id/reviews | yes | true no-mock HTTP | API_tests/test_reviews_notifications.sh | POST call at test_reviews_notifications.sh:62 |
| PATCH /api/collectibles/:id/hide | yes | true no-mock HTTP | API_tests/test_collectibles.sh | PATCH call at test_collectibles.sh:206 |
| PATCH /api/collectibles/:id/publish | yes | true no-mock HTTP | API_tests/test_collectibles.sh | PATCH call at test_collectibles.sh:215 |
| POST /api/orders | yes | true no-mock HTTP | API_tests/test_orders.sh | POST call at test_orders.sh:69 |
| GET /api/orders | yes | true no-mock HTTP | API_tests/test_orders.sh | GET call at test_orders.sh:146 |
| GET /api/orders/:id | yes | true no-mock HTTP | API_tests/test_orders.sh | GET call at test_orders.sh:134 |
| POST /api/orders/:id/confirm | yes | true no-mock HTTP | API_tests/test_orders.sh | POST call at test_orders.sh:164 |
| POST /api/orders/:id/process | yes | true no-mock HTTP | API_tests/test_orders.sh | POST call at test_orders.sh:174 |
| POST /api/orders/:id/complete | yes | true no-mock HTTP | API_tests/test_orders.sh | POST call at test_orders.sh:184 |
| POST /api/orders/:id/cancel | yes | true no-mock HTTP | API_tests/test_orders.sh | POST call at test_orders.sh:219 |
| POST /api/orders/:id/refund | yes | true no-mock HTTP | API_tests/test_order_advanced.sh | POST call at test_order_advanced.sh:184 |
| POST /api/orders/:id/arbitration | yes | true no-mock HTTP | API_tests/test_order_advanced.sh | POST call at test_order_advanced.sh:223 |
| PATCH /api/orders/:id/fulfillment | yes | true no-mock HTTP | API_tests/test_order_advanced.sh | PATCH call at test_order_advanced.sh:126 |
| GET /api/orders/:orderId/messages | yes | true no-mock HTTP | API_tests/test_messages.sh | GET call at test_messages.sh:112 |
| POST /api/orders/:orderId/messages | yes | true no-mock HTTP | API_tests/test_messages.sh | POST call at test_messages.sh:93 |
| GET /api/messages/:messageId/attachment | yes | true no-mock HTTP | API_tests/test_reviews_notifications.sh | GET call at test_reviews_notifications.sh:210 |
| GET /api/notifications | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:66 |
| PATCH /api/notifications/:id/read | yes | true no-mock HTTP | API_tests/test_reviews_notifications.sh | PATCH call at test_reviews_notifications.sh:143 |
| POST /api/notifications/read-all | yes | true no-mock HTTP | API_tests/test_notifications.sh | POST call at test_notifications.sh:73 |
| POST /api/notifications/:id/retry | yes | true no-mock HTTP | API_tests/test_high_risk_integration.sh | POST call at test_high_risk_integration.sh:531 |
| GET /api/notifications/preferences | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:80 |
| PUT /api/notifications/preferences | yes | true no-mock HTTP | API_tests/test_notifications.sh | PUT call at test_notifications.sh:88 |
| GET /api/analytics/funnel | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:221 |
| GET /api/analytics/retention | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:228 |
| GET /api/analytics/content-performance | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:235 |
| POST /api/ab-tests | yes | true no-mock HTTP | API_tests/test_consistency.sh | POST call at test_consistency.sh:89 |
| GET /api/ab-tests | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:285 |
| GET /api/ab-tests/:id | yes | true no-mock HTTP | API_tests/test_abtest_lifecycle.sh | GET call at test_abtest_lifecycle.sh:132 |
| PATCH /api/ab-tests/:id | yes | true no-mock HTTP | API_tests/test_abtest_lifecycle.sh | PATCH call at test_abtest_lifecycle.sh:160 |
| POST /api/ab-tests/:id/complete | yes | true no-mock HTTP | API_tests/test_abtest_lifecycle.sh | POST call at test_abtest_lifecycle.sh:176 |
| POST /api/ab-tests/:id/rollback | yes | true no-mock HTTP | API_tests/test_high_risk_integration.sh | POST call at test_high_risk_integration.sh:214 |
| GET /api/ab-tests/assignments | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:292 |
| GET /api/ab-tests/registry | yes | true no-mock HTTP | API_tests/test_abtest_lifecycle.sh | GET call at test_abtest_lifecycle.sh:61 |
| GET /api/admin/ip-rules | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:260 |
| POST /api/admin/ip-rules | yes | true no-mock HTTP | API_tests/test_admin_ops.sh | POST call at test_admin_ops.sh:61 |
| DELETE /api/admin/ip-rules/:id | yes | true no-mock HTTP | API_tests/test_admin_ops.sh | DELETE call at test_admin_ops.sh:134 |
| GET /api/admin/metrics | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:274 |
| GET /api/admin/anomalies | yes | true no-mock HTTP | API_tests/test_notifications.sh | GET call at test_notifications.sh:267 |
| PATCH /api/admin/anomalies/:id/acknowledge | yes | true no-mock HTTP | API_tests/test_admin_ops.sh | PATCH call at test_admin_ops.sh:180 |

### API Test Classification
1. True no-mock HTTP
- API_tests/test_00_setup.sh
- API_tests/test_abtest_lifecycle.sh
- API_tests/test_admin_ops.sh
- API_tests/test_auth.sh
- API_tests/test_collectibles.sh
- API_tests/test_consistency.sh
- API_tests/test_e2e_hardening.sh
- API_tests/test_high_risk_integration.sh
- API_tests/test_login_lockout.sh
- API_tests/test_messages.sh
- API_tests/test_notifications.sh
- API_tests/test_order_advanced.sh
- API_tests/test_orders.sh
- API_tests/test_reviews_notifications.sh
- API_tests/test_security.sh
- API_tests/test_user_mgmt.sh

2. HTTP with mocking
- none found by static inspection.

3. Non-HTTP (unit/integration without HTTP)
- unit_tests/*.test.js
- backend/internal/**/*_test.go
- frontend/src/**/*.test.ts and frontend/src/**/*.test.tsx
- e2e/*.e2e.ts (Playwright browser e2e)

### Mock Detection
Search scope: API_tests, unit_tests, frontend/src, backend/internal.

Markers checked:
- jest.mock
- vi.mock
- sinon.stub
- mock
- stub

Findings:
- none found by static inspection.

### Coverage Summary
- Total endpoints: 58
- Endpoints with HTTP tests: 58
- Endpoints with true no-mock tests: 58
- HTTP coverage: 58/58 = 100.00%
- True API coverage: 58/58 = 100.00%

### Unit Test Summary

#### Backend Unit Tests
Primary backend test files detected (representative):
- repo/backend/internal/handler/handler_integration_test.go
- repo/backend/internal/handler/authorization_isolation_test.go
- repo/backend/internal/handler/attachment_filter_test.go
- repo/backend/internal/handler/message_body_validation_test.go
- repo/backend/internal/service/auth_lockout_test.go
- repo/backend/internal/service/notification_service_test.go
- repo/backend/internal/service/notification_mode_test.go
- repo/backend/internal/service/abtest_service_test.go
- repo/backend/internal/service/abtest_registry_test.go
- repo/backend/internal/service/authorization_test.go
- repo/backend/internal/service/collectible_service_test.go
- repo/backend/internal/service/pii_test.go
- repo/backend/internal/model/order_test.go
- repo/backend/internal/model/order_transition_test.go
- repo/backend/internal/model/order_oversell_test.go
- repo/backend/internal/model/collectible_test.go
- repo/backend/internal/middleware/csrf_test.go
- repo/backend/internal/middleware/logging_test.go
- repo/backend/internal/store/anomaly_query_test.go
- repo/backend/internal/dto/validation_test.go
- repo/backend/internal/dto/error_codes_test.go
- repo/backend/internal/dto/request_test.go
- repo/backend/internal/dto/dashboard_test.go
- repo/backend/internal/worker/notification_retry_test.go

Modules covered:
- controllers/handlers: present
- services: present
- repositories/stores: present
- auth/guards/middleware: present

Important backend modules NOT tested:
- none found at route coverage level; residual gaps are mainly assertion-depth related rather than missing endpoint paths.

#### Frontend Unit Tests (STRICT REQUIREMENT)
Frontend unit test evidence:
- frontend test files found under frontend/src:
  - frontend/src/components/shared/Modal.test.tsx
  - frontend/src/components/shared/DataTable.test.tsx
  - frontend/src/components/shared/MaskedField.test.tsx
  - frontend/src/components/shared/ProtectedRoute.test.tsx
  - frontend/src/pages/pages.test.ts
  - frontend/src/store/authStore.test.ts
  - frontend/src/store/notificationStore.test.ts
  - frontend/src/store/abStore.test.ts
  - frontend/src/api/client.test.ts
  - frontend/src/types/types.test.ts
- framework detected: Vitest (example import at frontend/src/store/authStore.test.ts:1).
- tests import/render real frontend modules/components:
  - imports in frontend/src/pages/pages.test.ts:2-7 and frontend/src/components/shared/Modal.test.tsx:4
  - render/interaction usage in frontend/src/components/shared/DataTable.test.tsx:3,45,52; frontend/src/components/shared/MaskedField.test.tsx:3,14,21; frontend/src/components/shared/ProtectedRoute.test.tsx:3,37,71; frontend/src/components/shared/Modal.test.tsx:3,18,29

Mandatory verdict:
- Frontend unit tests: PRESENT

Browser-level e2e evidence (static):
- Playwright config present at repo/e2e/playwright.config.ts:1-21.
- Browser journey specs present at:
  - repo/e2e/auth.e2e.ts:1-57
  - repo/e2e/catalog.e2e.ts:1-33
  - repo/e2e/notifications.e2e.ts:1-46
  - repo/e2e/order-journey.e2e.ts:1-79
- Browser navigation assertions are visible (example: page.goto in repo/e2e/auth.e2e.ts:5 and repo/e2e/order-journey.e2e.ts:6).

Important frontend components/modules NOT comprehensively tested:
- frontend/src/hooks/*
- some deep error-state and edge-case page workflows remain less represented than happy-paths

### Cross-Layer Observation
- Backend/API route coverage is complete.
- Frontend has real component render/interaction tests.
- Browser-level FE<->BE e2e suites are now present (Playwright under repo/e2e).
- Confidence profile improves to API + frontend-unit + browser-e2e layers.

### API Observability Check
Strengths:
- clear method + path evidence in API tests (example: API_tests/test_user_mgmt.sh:73; API_tests/test_orders.sh:164).
- request payload/header details are present in many scripts.
- status and selected response assertions are present across suites.

Weaknesses:
- some checks are still status-centric and do not fully validate response contracts.

Verdict: medium-high.

### Test Quality & Sufficiency
- Success paths: strong.
- Failure paths: strong (RBAC, UUID validation, lockout, rollback, arbitration/refund cases).
- Edge cases: strong for advanced order and notification behavior.
- Validation/auth boundaries: strong at API level.
- Integration boundaries: strong via Docker-based API test harness.

run_tests.sh check:
- Docker-based orchestration is present in repo/run_tests.sh.
- helper packages are installed inside test container via apk add (repo/run_tests.sh:126,173,196), not host-level runtime setup.

### End-to-End Expectations
- Fullstack expectation: browser-level FE<->BE flows.
- Observed: explicit browser UI e2e suite exists under repo/e2e with auth/catalog/notifications/order-journey coverage.
- Remaining caveat: static inspection cannot confirm runtime pass/fail stability.

### Tests Check
- API suites are comprehensive and route-complete.
- True no-mock HTTP coverage is complete.
- Frontend unit tests are present under strict rules.
- Browser-level FE<->BE e2e suites are present by static evidence.
- Main remaining risk: runtime reliability is not validated in static-only mode.

### Test Coverage Score (0-100)
- Score: 96/100

### Score Rationale
- + 58/58 endpoint true HTTP coverage.
- + broad negative/security/lifecycle coverage.
- + no static evidence of mocking.
- + frontend has render/interaction component tests.
- + browser-level Playwright e2e coverage exists for critical journeys.
- - static-only audit cannot validate runtime pass/fail reliability.

### Key Gaps
- Some deeper page-level edge/failure workflows remain less represented than happy paths.
- Static-only method cannot verify runtime flakiness, timing, or infra-sensitive failures.

### Confidence & Assumptions
- Confidence: high for endpoint inventory and endpoint-level mapping.
- Confidence: medium-high for depth conclusions (static-only, not executed).
- Assumption: router.Setup in repo/backend/internal/router/router.go is the complete API registration source.

### Test Coverage Verdict
- PASS

---

## 2. README Audit

### README Location
- Required file repo/README.md: present.

### Hard Gate Evaluation

1. Formatting
- PASS
- Evidence: structured markdown sections and tables throughout repo/README.md.

2. Startup Instructions (Backend/Fullstack)
- PASS
- Evidence: exact token docker-compose up at repo/README.md:17 and repo/README.md:47.

3. Access Method
- PASS
- Evidence: URL/port table at repo/README.md:22-30.

4. Verification Method
- PASS
- Evidence: explicit verification steps at repo/README.md:47-53.

5. Environment Rules (STRICT)
- PASS
- Evidence: no npm/pip/apt/manual DB setup instruction; README states no manual setup for tests at repo/README.md:181.

6. Demo Credentials (Conditional)
- PASS
- Evidence: explicit username/password/role matrix at repo/README.md:36-41.

### Engineering Quality
- Tech stack clarity: strong (repo/README.md:5-13)
- Architecture explanation: good
- Testing instructions: clear and explicit across unit/API/E2E (repo/README.md:175-187)
- Security/roles explanation: good (repo/README.md:43, repo/README.md:151-163)
- Workflow clarity: good
- Presentation quality: strong

### High Priority Issues
- none.

### Medium Priority Issues
- none.

### Low Priority Issues
- none material.

### Hard Gate Failures
- none.

### README Verdict
- PASS

---

## Final Verdicts
- Test Coverage Audit: PASS
- README Audit: PASS

Overall combined status: PASS
