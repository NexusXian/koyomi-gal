ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS editor_mode VARCHAR(20) NOT NULL DEFAULT 'plain';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'posts_editor_mode_check'
          AND conrelid = 'posts'::regclass
    ) THEN
        ALTER TABLE posts
            ADD CONSTRAINT posts_editor_mode_check
            CHECK (editor_mode IN ('plain', 'markdown'));
    END IF;
END
$$;
