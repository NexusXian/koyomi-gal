ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_type_check;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_type_check CHECK (type IN (
        'post_commented', 'comment_replied', 'post_liked', 'comment_liked',
        'galgame_submitted', 'galgame_approved', 'galgame_rejected',
        'resource_submitted', 'resource_approved', 'resource_rejected', 'resource_hidden',
        'resource_reported', 'report_resolved', 'report_rejected',
        'post_moderated', 'comment_moderated', 'system'
    ));
