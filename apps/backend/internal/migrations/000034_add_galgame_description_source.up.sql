ALTER TABLE galgames
    ADD COLUMN description_source VARCHAR(32) NOT NULL DEFAULT '';

-- Backfill the source of existing descriptions from source_type:
-- 1 = VNDB import, 2 = Bangumi import. Mixed (3) and manual (0) rows cannot
-- be identified reliably, so they stay unknown and a later higher-priority
-- enrichment may refresh them. Never guess 'manual' for existing data.
UPDATE galgames SET description_source = 'vndb'
    WHERE description <> '' AND source_type = 1;

UPDATE galgames SET description_source = 'bangumi'
    WHERE description <> '' AND source_type = 2;

UPDATE galgames SET description_source = 'unknown'
    WHERE description <> '' AND source_type NOT IN (1, 2);

ALTER TABLE galgames
    ADD CONSTRAINT galgames_description_source_check
        CHECK (description_source IN ('', 'unknown', 'vndb', 'bangumi', 'manual'));
