ALTER TABLE articles
    DROP CONSTRAINT IF EXISTS articles_editor_mode_check;

ALTER TABLE articles
    DROP COLUMN IF EXISTS editor_mode;
