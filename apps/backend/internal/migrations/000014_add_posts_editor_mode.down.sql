ALTER TABLE posts
    DROP CONSTRAINT IF EXISTS posts_editor_mode_check;

ALTER TABLE posts
    DROP COLUMN IF EXISTS editor_mode;
