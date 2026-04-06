-- Remove the restored notification templates.
DELETE FROM notification_templates
 WHERE slug IN ('refund_approved', 'arbitration_opened', 'review_posted');
