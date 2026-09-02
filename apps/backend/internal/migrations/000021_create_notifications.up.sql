CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    recipient_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    category VARCHAR(32) NOT NULL
        CHECK (category IN ('interaction', 'review', 'moderation', 'system')),
    type VARCHAR(64) NOT NULL
        CHECK (type IN (
            'post_commented', 'comment_replied', 'post_liked', 'comment_liked',
            'galgame_submitted', 'galgame_approved', 'galgame_rejected',
            'resource_submitted', 'resource_approved', 'resource_rejected', 'resource_hidden',
            'resource_reported', 'report_resolved', 'report_rejected',
            'post_moderated', 'comment_moderated', 'system'
        )),
    entity_type VARCHAR(32),
    entity_id BIGINT,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    target_url VARCHAR(512),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient_created
    ON notifications (recipient_id, created_at DESC, id DESC);

CREATE INDEX idx_notifications_recipient_unread
    ON notifications (recipient_id, is_read, created_at DESC);

CREATE INDEX idx_notifications_entity
    ON notifications (entity_type, entity_id);
