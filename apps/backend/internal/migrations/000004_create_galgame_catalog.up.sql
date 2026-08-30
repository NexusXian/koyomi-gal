CREATE TABLE developers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL DEFAULT '',
    slug VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    website TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX developers_slug_unique ON developers (slug);
CREATE INDEX idx_developers_name ON developers (name);

CREATE TABLE galgames (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    original_title VARCHAR(255) NOT NULL DEFAULT '',
    romaji_title VARCHAR(255) NOT NULL DEFAULT '',
    slug VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    cover_url TEXT NOT NULL DEFAULT '',
    banner_url TEXT NOT NULL DEFAULT '',
    developer_id BIGINT,
    release_date DATE,
    age_rating SMALLINT NOT NULL DEFAULT 0,
    status SMALLINT NOT NULL DEFAULT 0,
    rating_average NUMERIC(3,2) NOT NULL DEFAULT 0,
    rating_count BIGINT NOT NULL DEFAULT 0,
    favorite_count BIGINT NOT NULL DEFAULT 0,
    resource_count BIGINT NOT NULL DEFAULT 0,
    post_count BIGINT NOT NULL DEFAULT 0,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_galgames_developer
        FOREIGN KEY (developer_id) REFERENCES developers(id) ON DELETE SET NULL,
    CONSTRAINT fk_galgames_created_by
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX galgames_slug_unique ON galgames (slug);
CREATE INDEX idx_galgames_developer_id ON galgames (developer_id);
CREATE INDEX idx_galgames_release_date ON galgames (release_date);
CREATE INDEX idx_galgames_status ON galgames (status);
CREATE INDEX idx_galgames_rating ON galgames (rating_average DESC);

CREATE TABLE galgame_aliases (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL,
    alias VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_galgame_aliases_galgame
        FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX galgame_aliases_unique ON galgame_aliases (galgame_id, alias);
CREATE INDEX idx_galgame_aliases_alias ON galgame_aliases (alias);

CREATE TABLE tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX tags_name_unique ON tags (name);
CREATE UNIQUE INDEX tags_slug_unique ON tags (slug);

CREATE TABLE galgame_tags (
    galgame_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_galgame_tags_galgame
        FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE,
    CONSTRAINT fk_galgame_tags_tag
        FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX galgame_tags_unique ON galgame_tags (galgame_id, tag_id);
CREATE INDEX idx_galgame_tags_tag_id ON galgame_tags (tag_id);
