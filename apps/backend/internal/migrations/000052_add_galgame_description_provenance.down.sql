ALTER TABLE galgame_descriptions
    DROP COLUMN IF EXISTS upstream_source,
    DROP COLUMN IF EXISTS source_external_id;
