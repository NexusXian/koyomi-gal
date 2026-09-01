CREATE TABLE banners (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    subtitle VARCHAR(500) NOT NULL DEFAULT '',
    image_url TEXT NOT NULL,
    link_type VARCHAR(32) NOT NULL DEFAULT 'none',
    link_value TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    start_at TIMESTAMPTZ,
    end_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT banners_link_type_check
        CHECK (link_type IN ('none', 'url', 'galgame', 'post', 'news')),
    CONSTRAINT banners_schedule_check
        CHECK (start_at IS NULL OR end_at IS NULL OR start_at < end_at),
    CONSTRAINT banners_link_value_check
        CHECK ((link_type = 'none' AND link_value = '') OR (link_type <> 'none' AND link_value <> ''))
);

CREATE INDEX idx_banners_public_order
    ON banners (sort_order DESC, id DESC)
    WHERE is_active = TRUE;
CREATE INDEX idx_banners_schedule ON banners (start_at, end_at);
