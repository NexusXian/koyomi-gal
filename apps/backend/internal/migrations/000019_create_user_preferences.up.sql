CREATE TABLE user_preferences (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    background_source VARCHAR(16) NOT NULL DEFAULT 'none'
        CHECK (background_source IN ('none', 'preset', 'custom')),
    background_asset_id BIGINT REFERENCES image_assets(id) ON DELETE SET NULL,
    background_preset VARCHAR(64),
    background_opacity REAL NOT NULL DEFAULT 0.35
        CHECK (background_opacity BETWEEN 0 AND 1),
    background_blur REAL NOT NULL DEFAULT 0
        CHECK (background_blur BETWEEN 0 AND 20),
    background_position VARCHAR(64) NOT NULL DEFAULT 'center center',
    background_size VARCHAR(16) NOT NULL DEFAULT 'cover'
        CHECK (background_size IN ('cover', 'contain')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_preferences_source_custom_check
        CHECK (background_source <> 'custom' OR background_asset_id IS NOT NULL),
    CONSTRAINT user_preferences_source_preset_check
        CHECK (background_source <> 'preset' OR background_preset IS NOT NULL)
);
