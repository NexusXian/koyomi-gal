DROP TABLE IF EXISTS import_jobs;
DROP TABLE IF EXISTS galgame_external_sources;

ALTER TABLE galgames
    DROP COLUMN IF EXISTS metadata_updated_at,
    DROP COLUMN IF EXISTS source_type,
    DROP COLUMN IF EXISTS length_minutes,
    DROP COLUMN IF EXISTS original_language;
