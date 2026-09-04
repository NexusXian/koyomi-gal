ALTER TABLE user_preferences
DROP COLUMN IF EXISTS sensitive_cover_mode;

DROP INDEX IF EXISTS idx_galgames_age_rating;
DROP INDEX IF EXISTS idx_galgames_cover_sensitive;

ALTER TABLE galgames
DROP COLUMN IF EXISTS cover_sensitive;
