CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) UNIQUE NOT NULL,
    title_template TEXT NOT NULL,
    body_template TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO notification_templates (slug, title_template, body_template) VALUES
    ('order_confirmed', 'Order Confirmed', 'Your order {{.OrderID}} has been confirmed by the seller.'),
    ('order_processing', 'Order Processing', 'Your order {{.OrderID}} is now being processed.'),
    ('order_completed', 'Order Completed', 'Your order {{.OrderID}} has been completed. Enjoy your collectible!'),
    ('order_cancelled', 'Order Cancelled', 'Your order {{.OrderID}} has been cancelled. Reason: {{.Reason}}'),
    ('refund_approved', 'Refund Approved', 'Your refund for order {{.OrderID}} has been approved.'),
    ('arbitration_opened', 'Arbitration Opened', 'An arbitration case has been opened for order {{.OrderID}}. A compliance analyst will review your case.'),
    ('review_posted', 'Review Posted', 'A review has been posted for your collectible {{.CollectibleTitle}}.');

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES notification_templates(id),
    params JSONB,
    rendered_title TEXT NOT NULL,
    rendered_body TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'delivered' CHECK (status IN ('delivered', 'failed', 'pending')),
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_retry ON notifications(status, next_retry_at) WHERE status = 'failed' AND retry_count < max_retries;

CREATE TABLE notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    preferences JSONB NOT NULL DEFAULT '{}'
);
