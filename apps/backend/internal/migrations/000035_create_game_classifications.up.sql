CREATE TABLE game_classifications (
    id BIGSERIAL PRIMARY KEY,
    game_id BIGINT NOT NULL,
    classification VARCHAR(16) NOT NULL DEFAULT '',
    confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    conflict BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(24) NOT NULL DEFAULT 'queued',
    model VARCHAR(128) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    reviewer_id BIGINT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_game_classifications_game
        FOREIGN KEY (game_id) REFERENCES galgames(id) ON DELETE CASCADE,
    CONSTRAINT fk_game_classifications_reviewer
        FOREIGN KEY (reviewer_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_game_classifications_game_id_created_at
    ON game_classifications (game_id, created_at DESC);
CREATE INDEX idx_game_classifications_status_created_at
    ON game_classifications (status, created_at DESC);
CREATE INDEX idx_game_classifications_review_queue
    ON game_classifications (classification, confidence DESC, status);

CREATE TABLE game_classification_evidences (
    id BIGSERIAL PRIMARY KEY,
    classification_id BIGINT NOT NULL,
    source_type VARCHAR(32) NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    evidence TEXT NOT NULL DEFAULT '',
    weight INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_game_classification_evidences_classification
        FOREIGN KEY (classification_id) REFERENCES game_classifications(id) ON DELETE CASCADE
);

CREATE INDEX idx_game_classification_evidences_classification_id
    ON game_classification_evidences (classification_id, id);
