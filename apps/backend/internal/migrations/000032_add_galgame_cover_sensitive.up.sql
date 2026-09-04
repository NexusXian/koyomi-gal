ALTER TABLE galgames
ADD COLUMN cover_sensitive BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_galgames_cover_sensitive ON galgames (cover_sensitive);
CREATE INDEX idx_galgames_age_rating ON galgames (age_rating);

ALTER TABLE user_preferences
ADD COLUMN sensitive_cover_mode VARCHAR(16) NOT NULL DEFAULT 'blur'
    CHECK (sensitive_cover_mode IN ('blur', 'show'));
