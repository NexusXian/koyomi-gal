CREATE TABLE image_assets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    object_key VARCHAR(512) NOT NULL UNIQUE,
    original_name VARCHAR(255) NOT NULL DEFAULT '',
    mime_type VARCHAR(100) NOT NULL,
    extension VARCHAR(20) NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    width INTEGER,
    height INTEGER,
    category VARCHAR(50) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT image_assets_status_check
        CHECK (status IN (0, 1, 2, 3)),
    CONSTRAINT image_assets_deleted_check
        CHECK ((status = 2) = (deleted_at IS NOT NULL))
);

CREATE INDEX idx_image_assets_user_id ON image_assets(user_id);
CREATE INDEX idx_image_assets_category ON image_assets(category);
CREATE INDEX idx_image_assets_created_at ON image_assets(created_at);
CREATE INDEX idx_image_assets_pending_created_at
    ON image_assets(created_at)
    WHERE status = 0;
