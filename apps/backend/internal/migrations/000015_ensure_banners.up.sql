ALTER TABLE banners
    ADD COLUMN IF NOT EXISTS title VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS subtitle VARCHAR(500) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS link_type VARCHAR(32) NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS link_value TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS start_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS end_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE banners
    ALTER COLUMN title DROP DEFAULT,
    ALTER COLUMN image_url DROP DEFAULT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'banners_link_type_check'
          AND conrelid = 'banners'::regclass
    ) THEN
        ALTER TABLE banners
            ADD CONSTRAINT banners_link_type_check
            CHECK (link_type IN ('none', 'url', 'galgame', 'post', 'news'));
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'banners_link_value_check'
          AND conrelid = 'banners'::regclass
    ) THEN
        ALTER TABLE banners
            ADD CONSTRAINT banners_link_value_check
            CHECK ((link_type = 'none' AND link_value = '') OR (link_type <> 'none' AND link_value <> ''));
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_banners_public_order
    ON banners (sort_order DESC, id DESC)
    WHERE is_active = TRUE;
