ALTER TABLE galgames
    ADD COLUMN original_language VARCHAR(16) NOT NULL DEFAULT '',
    ADD COLUMN length_minutes INTEGER,
    ADD COLUMN source_type SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN metadata_updated_at TIMESTAMPTZ;

CREATE TABLE galgame_external_sources (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL,
    source VARCHAR(32) NOT NULL,
    external_id VARCHAR(128) NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    external_rating NUMERIC(4,2),
    external_rating_count INTEGER,
    raw_metadata JSONB,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_galgame_external_sources_galgame
        FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX galgame_external_sources_source_external_id_unique
    ON galgame_external_sources (source, external_id);
CREATE INDEX idx_galgame_external_sources_galgame_id
    ON galgame_external_sources (galgame_id);

CREATE TABLE import_jobs (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(32) NOT NULL,
    job_type VARCHAR(32) NOT NULL,
    status SMALLINT NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    processed_count INTEGER NOT NULL DEFAULT 0,
    created_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    params JSONB,
    error_message TEXT NOT NULL DEFAULT '',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    CONSTRAINT fk_import_jobs_created_by
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_import_jobs_created_at ON import_jobs (created_at DESC);
CREATE INDEX idx_import_jobs_status ON import_jobs (status);
