DROP TABLE IF EXISTS external_match_candidates;

ALTER TABLE import_jobs
    DROP COLUMN IF EXISTS stats;
