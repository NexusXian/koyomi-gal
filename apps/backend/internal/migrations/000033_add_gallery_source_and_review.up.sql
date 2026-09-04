-- Existing gallery images are already live: backfill them as published (1)
-- first, then flip the default so every future row starts as pending (0).
ALTER TABLE galgame_gallery_images
    ALTER COLUMN asset_id DROP NOT NULL,
    ADD COLUMN source_type SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN external_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN status SMALLINT NOT NULL DEFAULT 1,
    ADD COLUMN reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN reviewed_at TIMESTAMPTZ,
    ADD COLUMN reject_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE galgame_gallery_images
    ADD CONSTRAINT galgame_gallery_images_source_type_check
        CHECK (source_type IN (0, 1)),
    ADD CONSTRAINT galgame_gallery_images_status_check
        CHECK (status IN (0, 1, 2)),
    ADD CONSTRAINT galgame_gallery_images_source_check
        CHECK (
            (source_type = 0 AND asset_id IS NOT NULL AND external_url = '')
            OR (source_type = 1 AND asset_id IS NULL AND external_url <> '')
        );

ALTER TABLE galgame_gallery_images
    ALTER COLUMN status SET DEFAULT 0;

CREATE UNIQUE INDEX uk_galgame_gallery_images_external_url
    ON galgame_gallery_images(galgame_id, external_url)
    WHERE external_url <> '';

CREATE INDEX idx_galgame_gallery_images_status
    ON galgame_gallery_images(status, created_at DESC);
