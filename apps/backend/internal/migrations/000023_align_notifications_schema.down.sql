UPDATE notifications SET target_url = '' WHERE target_url IS NULL;

ALTER TABLE notifications
    ALTER COLUMN target_url TYPE TEXT,
    ALTER COLUMN target_url SET NOT NULL,
    ALTER COLUMN target_url SET DEFAULT '';

DROP INDEX IF EXISTS idx_notifications_entity;

DROP INDEX IF EXISTS idx_notifications_recipient_unread;
CREATE INDEX idx_notifications_recipient_unread
    ON notifications (recipient_id) WHERE read_at IS NULL;

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_type_check;

ALTER TABLE notifications
    DROP COLUMN IF EXISTS entity_type,
    DROP COLUMN IF EXISTS entity_id;
