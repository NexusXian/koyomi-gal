CREATE TABLE work_contributions (
    id BIGSERIAL PRIMARY KEY,
    target_type VARCHAR(32) NOT NULL,
    target_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    action VARCHAR(32) NOT NULL,
    source_type VARCHAR(32),
    source_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT work_contributions_target_type_check
        CHECK (target_type IN ('galgame', 'novel')),
    CONSTRAINT work_contributions_source_pair_check CHECK (
        (source_type IS NULL AND source_id IS NULL) OR
        (source_type IS NOT NULL AND source_id IS NOT NULL)
    )
);

CREATE INDEX work_contributions_target_idx ON work_contributions (target_type, target_id);
CREATE INDEX work_contributions_user_id_idx ON work_contributions (user_id);
CREATE UNIQUE INDEX work_contributions_source_unique
    ON work_contributions (source_type, source_id)
    WHERE source_type IS NOT NULL AND source_id IS NOT NULL;

INSERT INTO work_contributions (
    target_type, target_id, user_id, action, source_type, source_id, created_at
)
SELECT 'galgame', galgame_id, user_id, action, source_type, source_id, created_at
FROM galgame_contributions;

DROP TABLE galgame_contributions;
