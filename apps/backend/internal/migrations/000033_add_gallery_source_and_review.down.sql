DROP INDEX IF EXISTS idx_galgame_gallery_images_status;
DROP INDEX IF EXISTS uk_galgame_gallery_images_external_url;

ALTER TABLE galgame_gallery_images
    DROP CONSTRAINT IF EXISTS galgame_gallery_images_source_check,
    DROP CONSTRAINT IF EXISTS galgame_gallery_images_status_check,
    DROP CONSTRAINT IF EXISTS galgame_gallery_images_source_type_check;

ALTER TABLE galgame_gallery_images
    DROP COLUMN IF EXISTS reject_reason,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS external_url,
    DROP COLUMN IF EXISTS source_type;

-- Reverting requires every remaining row to be asset-backed; external-source
-- rows have no asset to fall back to and must be deleted first.
DELETE FROM galgame_gallery_images WHERE asset_id IS NULL;

ALTER TABLE galgame_gallery_images
    ALTER COLUMN asset_id SET NOT NULL;
