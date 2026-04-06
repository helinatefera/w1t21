-- Restore notification templates for refund, arbitration, and review features.
-- These were originally seeded in 005 and removed in 013 because no emitters
-- existed at the time. Business-flow emitters now exist in the service layer.

INSERT INTO notification_templates (slug, title_template, body_template) VALUES
    ('refund_approved', 'Refund Approved',
     'Your refund for order {{.OrderID}} has been approved. Reason: {{.Reason}}'),
    ('arbitration_opened', 'Arbitration Opened',
     'An arbitration case has been opened for order {{.OrderID}}. A compliance analyst will review your case.'),
    ('review_posted', 'Review Posted',
     'A review has been posted for your collectible {{.CollectibleTitle}}.')
ON CONFLICT (slug) DO NOTHING;
