CREATE TABLE resources (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL,
    uploader_id BIGINT,
    title VARCHAR(255) NOT NULL,
    resource_type SMALLINT NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_resources_galgame
        FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE,
    CONSTRAINT fk_resources_uploader
        FOREIGN KEY (uploader_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT resources_type_range CHECK (resource_type >= 0 AND resource_type <= 6),
    CONSTRAINT resources_status_range CHECK (status >= 0 AND status <= 3)
);

CREATE INDEX idx_resources_galgame_id ON resources (galgame_id);
CREATE INDEX idx_resources_uploader_id ON resources (uploader_id);
CREATE INDEX idx_resources_status ON resources (status);

CREATE TABLE resource_links (
    id BIGSERIAL PRIMARY KEY,
    resource_id BIGINT NOT NULL,
    url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_resource_links_resource
        FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE
);

CREATE INDEX idx_resource_links_resource_id ON resource_links (resource_id);
