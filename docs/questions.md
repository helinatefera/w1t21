## 1. User Registration vs Predefined Accounts
**Question:** The prompt does not mention whether users can self register or must be created by an administrator.  
**Assumption:** Admin only creation to maintain closed LAN control.  
**Solution:** Admin user management with password reset flow.

## 2. Password Storage & Hashing
**Question:** The prompt mentions encryption for sensitive data but does not clarify how user passwords are stored for authentication.  
**Assumption:** Passwords are hashed (bcrypt/Argon2), not encrypted with the application key.  
**Solution:** Store bcrypt hashes in PostgreSQL; derive separate encryption keys if needed.

## 3. Encryption Scope & Key Management
**Question:** The prompt mentions “sensitive data such as phone numbers or government IDs” but does not specify which exact fields are considered sensitive, nor how the application managed key is stored and rotated.  
**Assumption:** All PII and message attachments encrypted with an app wide AES-256 key.  
**Solution:** Mark columns with `@Encrypted`, load key from encrypted config file at startup.

## 4. Session & JWT Lifecycle
**Question:** The prompt mentions JWT or server sessions but does not clarify expiration times, refresh behavior, or logout rules.  
**Assumption:** 15 min access tokens + 7-day refresh tokens stored in httpOnly cookies.  
**Solution:** Refresh token rotation; clear refresh token from database on logout.

## 5. Rate Limiting Granularity
**Question:** The prompt specifies rate limiting for login attempts (10 per 15 min) but does not mention limits for other endpoints like order placement, messaging, or listing creation.  
**Assumption:** Order creation (30/min), messaging (20/min), listing creation (10/hr).  
**Solution:** Token bucket with per user and per IP keys; return 429 with Retry After.

## 6. Offline LAN Mode vs True Offline
**Question:** The prompt says “capable of running fully offline on a company LAN” but does not clarify whether this means no internet connection (backend still required) or no server at all (pure client side).  
**Assumption:** Air gapped LAN with backend (Go + PostgreSQL) still required.  
**Solution:** Docker Compose stack with vendored dependencies; React served as static files.

## 7. Multi Role Assignment
**Question:** The prompt defines multiple roles (Buyer, Seller, Administrator, Compliance Analyst) but does not mention whether a single user can hold more than one role.  
**Assumption:** Yes, roles are additive.  
**Solution:** RBAC with `user_roles` junction table; UI merges capabilities.

## 8. Collectible Listing Approval Workflow
**Question:** The prompt does not mention any moderation or approval step for listings created by Sellers.  
**Assumption:** Auto published but admins can hide/delete violating listings.  
**Solution:** Add `status` column (published/hidden) and admin moderation queue.

## 9. Order Fulfillment States
**Question:** The prompt mentions “track fulfillment” but does not define the possible order states or transition rules.  
**Assumption:** pending → confirmed → processing → completed; cancellations allowed only before shipping.  
**Solution:** Implement state machine with allowed transitions per role; notify on each change.

## 10. On-Device Pattern Detection
**Question:** The prompt says “blocks sensitive content such as phone numbers and SSNs using on device pattern detection” but does not clarify whether this applies to message text only, attachments only, or both.  
**Assumption:** Both, using regex for phone numbers and SSNs.  
**Solution:** Client side regex (plus optional OCR for images); block submission on match.

## 11. Notification Template Subscription
**Question:** The prompt mentions subscribing to templates (e.g., “status changes only” vs “all events”) but does not clarify whether templates are predefined by the system or user‑definable, nor how subscription preferences are stored.  
**Assumption:** Predefined system templates; users opt‑in/out per template.  
**Solution:** Store user preferences as JSON array of enabled template IDs.

## 12. A/B Test Conversion Metrics & Rollback
**Question:** The prompt mentions a “one click rollback … if conversion drops beyond a defined threshold” but does not specify which conversion metric (e.g., view‑to‑order rate) nor how the threshold is measured (absolute or relative).  
**Assumption:** View‑to‑order rate; relative drop threshold set by admin (e.g., 15%).  
**Solution:** Background job computes conversion every 5 min; auto rollback if treatment falls below control minus threshold.
