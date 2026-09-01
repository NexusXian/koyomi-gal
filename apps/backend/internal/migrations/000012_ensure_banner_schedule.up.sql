ALTER TABLE banners
    ADD COLUMN IF NOT EXISTS start_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS end_at TIMESTAMPTZ;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'banners_schedule_check'
          AND conrelid = 'banners'::regclass
    ) THEN
        ALTER TABLE banners
            ADD CONSTRAINT banners_schedule_check
            CHECK (start_at IS NULL OR end_at IS NULL OR start_at < end_at);
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_banners_schedule ON banners (start_at, end_at);
