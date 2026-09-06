CREATE TABLE characters (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL CHECK (BTRIM(name) <> ''),
    original_name VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    image_url VARCHAR(2048) NOT NULL DEFAULT '',
    gender VARCHAR(50) NOT NULL DEFAULT '',
    birthday VARCHAR(50) NOT NULL DEFAULT '',
    blood_type VARCHAR(50) NOT NULL DEFAULT '',
    height INT NOT NULL DEFAULT 0 CHECK (height >= 0),
    source VARCHAR(50) NOT NULL DEFAULT '',
    source_id VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_characters_source_source_id ON characters(source, source_id)
    WHERE source <> '' AND source_id <> '';

CREATE TABLE galgame_characters (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL REFERENCES galgames(id) ON DELETE CASCADE,
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    role SMALLINT NOT NULL DEFAULT 0 CHECK (role IN (0, 1, 2, 3, 4)),
    spoiler_level SMALLINT NOT NULL DEFAULT 0 CHECK (spoiler_level IN (0, 1, 2)),
    appearance_spoiler BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT NOT NULL DEFAULT '',
    spoiler_description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_galgame_characters_galgame_character UNIQUE (galgame_id, character_id)
);

CREATE INDEX idx_galgame_characters_galgame_id ON galgame_characters(galgame_id);
CREATE INDEX idx_galgame_characters_character_id ON galgame_characters(character_id);
