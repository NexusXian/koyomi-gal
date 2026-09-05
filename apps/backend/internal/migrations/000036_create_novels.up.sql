CREATE TABLE novels (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    original_title VARCHAR(255) NOT NULL DEFAULT '',
    slug VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    cover_url TEXT NOT NULL DEFAULT '',
    author VARCHAR(255) NOT NULL DEFAULT '',
    illustrator VARCHAR(255) NOT NULL DEFAULT '',
    publisher VARCHAR(255) NOT NULL DEFAULT '',
    label VARCHAR(255) NOT NULL DEFAULT '',
    language VARCHAR(16) NOT NULL DEFAULT '',
    region VARCHAR(16) NOT NULL DEFAULT '',
    release_status VARCHAR(16) NOT NULL DEFAULT 'unknown',
    first_release_date DATE,
    age_rating SMALLINT NOT NULL DEFAULT 0,
    is_cover_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    official_website TEXT NOT NULL DEFAULT '',
    resource_count BIGINT NOT NULL DEFAULT 0,
    created_by BIGINT,
    status SMALLINT NOT NULL DEFAULT 0,
    reviewed_by BIGINT,
    reviewed_at TIMESTAMPTZ,
    reject_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_novels_created_by
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_novels_reviewed_by
        FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT novels_release_status_check
        CHECK (release_status IN ('ongoing', 'completed', 'hiatus', 'cancelled', 'unknown')),
    CONSTRAINT novels_status_range CHECK (status >= 0 AND status <= 3)
);

CREATE UNIQUE INDEX novels_slug_unique ON novels (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_novels_title ON novels (title);
CREATE INDEX idx_novels_original_title ON novels (original_title);
CREATE INDEX idx_novels_status ON novels (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_novels_author ON novels (author);
CREATE INDEX idx_novels_publisher ON novels (publisher);
CREATE INDEX idx_novels_label ON novels (label);
CREATE INDEX idx_novels_release_status ON novels (release_status);
CREATE INDEX idx_novels_language ON novels (language);
CREATE INDEX idx_novels_first_release_date ON novels (first_release_date);
CREATE INDEX idx_novels_created_by ON novels (created_by);

CREATE TABLE novel_volumes (
    id BIGSERIAL PRIMARY KEY,
    novel_id BIGINT NOT NULL,
    volume_number INT,
    title VARCHAR(255) NOT NULL DEFAULT '',
    original_title VARCHAR(255) NOT NULL DEFAULT '',
    cover_url TEXT NOT NULL DEFAULT '',
    isbn VARCHAR(20) NOT NULL DEFAULT '',
    release_date DATE,
    summary TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_by BIGINT,
    status SMALLINT NOT NULL DEFAULT 0,
    reviewed_by BIGINT,
    reviewed_at TIMESTAMPTZ,
    reject_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_novel_volumes_novel
        FOREIGN KEY (novel_id) REFERENCES novels(id) ON DELETE CASCADE,
    CONSTRAINT fk_novel_volumes_created_by
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_novel_volumes_reviewed_by
        FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT novel_volumes_status_range CHECK (status >= 0 AND status <= 3)
);

CREATE INDEX idx_novel_volumes_novel_sort ON novel_volumes (novel_id, sort_order);
CREATE INDEX idx_novel_volumes_status ON novel_volumes (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_novel_volumes_isbn ON novel_volumes (isbn);

CREATE TABLE novel_tags (
    novel_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_novel_tags_novel
        FOREIGN KEY (novel_id) REFERENCES novels(id) ON DELETE CASCADE,
    CONSTRAINT fk_novel_tags_tag
        FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX novel_tags_unique ON novel_tags (novel_id, tag_id);
CREATE INDEX idx_novel_tags_tag_id ON novel_tags (tag_id);

CREATE TABLE work_relations (
    id BIGSERIAL PRIMARY KEY,
    source_type VARCHAR(32) NOT NULL,
    source_id BIGINT NOT NULL,
    target_type VARCHAR(32) NOT NULL,
    target_id BIGINT NOT NULL,
    relation_type VARCHAR(32) NOT NULL,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_work_relations_created_by
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT work_relations_work_type_check
        CHECK (source_type IN ('galgame', 'novel') AND target_type IN ('galgame', 'novel')),
    CONSTRAINT work_relations_relation_type_check
        CHECK (relation_type IN ('adaptation', 'original', 'spin_off', 'sequel', 'prequel', 'same_series', 'related')),
    CONSTRAINT work_relations_no_self_check
        CHECK (NOT (source_type = target_type AND source_id = target_id))
);

CREATE UNIQUE INDEX work_relations_unique
    ON work_relations (source_type, source_id, target_type, target_id, relation_type);
CREATE INDEX idx_work_relations_source ON work_relations (source_type, source_id);
CREATE INDEX idx_work_relations_target ON work_relations (target_type, target_id);

CREATE TABLE external_mappings (
    id BIGSERIAL PRIMARY KEY,
    target_type VARCHAR(32) NOT NULL,
    target_id BIGINT NOT NULL,
    source VARCHAR(32) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    external_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT external_mappings_target_type_check
        CHECK (target_type IN ('galgame', 'novel'))
);

CREATE UNIQUE INDEX external_mappings_unique
    ON external_mappings (target_type, target_id, source, external_id);
CREATE INDEX idx_external_mappings_source ON external_mappings (source, external_id);
CREATE INDEX idx_external_mappings_target ON external_mappings (target_type, target_id);
