ALTER TABLE galgames
    DROP CONSTRAINT IF EXISTS galgames_description_source_check;

ALTER TABLE galgames
    DROP COLUMN description_source;
