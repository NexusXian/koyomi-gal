ALTER TABLE users
    ADD COLUMN avatar_asset_id BIGINT REFERENCES image_assets(id) ON DELETE SET NULL;
