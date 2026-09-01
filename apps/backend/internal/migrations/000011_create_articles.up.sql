CREATE TABLE articles (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    summary VARCHAR(500) NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    cover_url TEXT NOT NULL DEFAULT '',
    type VARCHAR(32) NOT NULL,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    view_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT articles_type_check
        CHECK (type IN ('announcement', 'news', 'event', 'update')),
    CONSTRAINT articles_view_count_check CHECK (view_count >= 0)
);

CREATE INDEX idx_articles_public_list
    ON articles (is_pinned DESC, published_at DESC, id DESC)
    WHERE is_published = TRUE AND published_at IS NOT NULL;
CREATE INDEX idx_articles_type_public_list
    ON articles (type, is_pinned DESC, published_at DESC, id DESC)
    WHERE is_published = TRUE AND published_at IS NOT NULL;
