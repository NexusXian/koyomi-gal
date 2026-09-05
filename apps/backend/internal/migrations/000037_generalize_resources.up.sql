ALTER TABLE resources
    ADD COLUMN target_type VARCHAR(32) NOT NULL DEFAULT 'galgame',
    ADD COLUMN target_id BIGINT;

UPDATE resources SET target_id = galgame_id;

ALTER TABLE resources ALTER COLUMN target_id SET NOT NULL;

ALTER TABLE resources DROP CONSTRAINT fk_resources_galgame;
ALTER TABLE resources DROP COLUMN galgame_id;

DROP INDEX IF EXISTS idx_resources_galgame_id;
CREATE INDEX idx_resources_target ON resources (target_type, target_id);

ALTER TABLE resources DROP CONSTRAINT resources_type_range;
ALTER TABLE resources
    ADD CONSTRAINT resources_type_range CHECK (resource_type >= 0 AND resource_type <= 12);
