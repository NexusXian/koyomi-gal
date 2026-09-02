ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS entity_type VARCHAR(32),
    ADD COLUMN IF NOT EXISTS entity_id BIGINT;

ALTER TABLE notifications
    ALTER COLUMN target_url TYPE VARCHAR(512),
    ALTER COLUMN target_url DROP NOT NULL,
    ALTER COLUMN target_url DROP DEFAULT;

UPDATE notifications SET target_url = NULL WHERE target_url = '';

CREATE INDEX IF NOT EXISTS idx_notifications_entity
    ON notifications (entity_type, entity_id);

DROP INDEX IF EXISTS idx_notifications_recipient_unread;
CREATE INDEX idx_notifications_recipient_unread
    ON notifications (recipient_id, is_read, created_at DESC);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'notifications_type_check'
    ) THEN
        ALTER TABLE notifications
            ADD CONSTRAINT notifications_type_check CHECK (type IN (
                'post_commented', 'comment_replied', 'post_liked', 'comment_liked',
                'galgame_submitted', 'galgame_approved', 'galgame_rejected',
                'resource_submitted', 'resource_approved', 'resource_rejected', 'resource_hidden',
                'resource_reported', 'report_resolved', 'report_rejected',
                'post_moderated', 'comment_moderated', 'system'
            ));
    END IF;
END $$;
