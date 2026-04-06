-- Remove notification templates that have no corresponding business-flow emitter.
-- These were seeded in 005 but no refund, arbitration, or review feature exists,
-- so they create a false impression of supported functionality.
--
-- Safety: delete only if no notifications reference them (FK on template_id).
-- If any notifications exist for these templates, the DELETE will fail due to
-- the FK constraint, which is the correct behaviour — it means the template was
-- somehow used and should not be silently removed.

DELETE FROM notification_templates
 WHERE slug IN ('refund_approved', 'arbitration_opened', 'review_posted');
