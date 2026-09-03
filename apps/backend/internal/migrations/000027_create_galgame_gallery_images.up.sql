CREATE TABLE galgame_gallery_images (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL REFERENCES galgames(id) ON DELETE CASCADE,
    asset_id BIGINT NOT NULL REFERENCES image_assets(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    image_type SMALLINT NOT NULL DEFAULT 0,
    is_spoiler BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT galgame_gallery_images_type_check
        CHECK (image_type IN (0, 1, 2, 3, 4))
);

CREATE INDEX idx_galgame_gallery_images_galgame_sort
    ON galgame_gallery_images(galgame_id, sort_order);
CREATE UNIQUE INDEX uk_galgame_gallery_images_galgame_asset
    ON galgame_gallery_images(galgame_id, asset_id);
