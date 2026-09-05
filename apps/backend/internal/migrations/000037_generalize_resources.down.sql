ALTER TABLE resources DROP CONSTRAINT resources_type_range;
ALTER TABLE resources
    ADD CONSTRAINT resources_type_range CHECK (resource_type >= 0 AND resource_type <= 6);

DROP INDEX IF EXISTS idx_resources_target;

ALTER TABLE resources
    ADD COLUMN galgame_id BIGINT;

UPDATE resources SET galgame_id = target_id WHERE target_type = 'galgame';

ALTER TABLE resources ALTER COLUMN galgame_id SET NOT NULL;
ALTER TABLE resources
    ADD CONSTRAINT fk_resources_galgame
    FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE;

ALTER TABLE resources DROP COLUMN target_id;
ALTER TABLE resources DROP COLUMN target_type;

CREATE INDEX idx_resources_galgame_id ON resources (galgame_id);
