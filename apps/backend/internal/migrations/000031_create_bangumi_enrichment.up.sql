ALTER TABLE import_jobs
    ADD COLUMN stats JSONB NOT NULL DEFAULT '{}';

CREATE TABLE external_match_candidates (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL,
    provider VARCHAR(32) NOT NULL,
    external_id VARCHAR(128) NOT NULL,
    confidence NUMERIC(5,4) NOT NULL,
    reasons JSONB NOT NULL DEFAULT '[]',
    preview JSONB,
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,
    reviewed_by BIGINT,
    CONSTRAINT fk_external_match_candidates_galgame
        FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE,
    CONSTRAINT fk_external_match_candidates_reviewer
        FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX external_match_candidates_identity_unique
    ON external_match_candidates (galgame_id, provider, external_id);
CREATE INDEX idx_external_match_candidates_status
    ON external_match_candidates (status, confidence DESC);
