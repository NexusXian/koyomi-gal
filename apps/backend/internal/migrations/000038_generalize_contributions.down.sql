CREATE TABLE galgame_contributions (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL REFERENCES galgames(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    action VARCHAR(32) NOT NULL,
    source_type VARCHAR(32),
    source_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT galgame_contributions_source_pair_check CHECK (
        (source_type IS NULL AND source_id IS NULL) OR
        (source_type IS NOT NULL AND source_id IS NOT NULL)
    )
);

CREATE INDEX galgame_contributions_galgame_id_idx
    ON galgame_contributions(galgame_id);
CREATE INDEX galgame_contributions_user_id_idx
    ON galgame_contributions(user_id);
CREATE INDEX galgame_contributions_galgame_user_idx
    ON galgame_contributions(galgame_id, user_id);
CREATE UNIQUE INDEX galgame_contributions_source_unique
    ON galgame_contributions(source_type, source_id)
    WHERE source_type IS NOT NULL AND source_id IS NOT NULL;

INSERT INTO galgame_contributions (
    galgame_id, user_id, action, source_type, source_id, created_at
)
SELECT target_id, user_id, action, source_type, source_id, created_at
FROM work_contributions
WHERE target_type = 'galgame';

DROP TABLE work_contributions;
