-- Re-insert the removed templates so the migration is reversible.
INSERT INTO notification_templates (slug, title_template, body_template) VALUES
    ('refund_approved', 'Refund Approved',
     'Your refund for order {{.OrderID}} has been approved.'),
    ('arbitration_opened', 'Arbitration Opened',
     'An arbitration case has been opened for order {{.OrderID}}. A compliance analyst will review your case.'),
    ('review_posted', 'Review Posted',
     'A review has been posted for your collectible {{.CollectibleTitle}}.')
ON CONFLICT (slug) DO NOTHING;
