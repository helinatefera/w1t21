ALTER TABLE notification_preferences
    ADD COLUMN subscription_mode VARCHAR(20) NOT NULL DEFAULT 'all_events'
    CHECK (subscription_mode IN ('status_only', 'all_events'));
