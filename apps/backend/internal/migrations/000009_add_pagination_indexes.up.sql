CREATE INDEX idx_resources_galgame_status_id
    ON resources (galgame_id, status, id DESC);

CREATE INDEX idx_comments_post_top_level_id
    ON comments (post_id, id)
    WHERE parent_id IS NULL;

CREATE INDEX idx_comments_parent_id_id
    ON comments (parent_id, id)
    WHERE parent_id IS NOT NULL;
