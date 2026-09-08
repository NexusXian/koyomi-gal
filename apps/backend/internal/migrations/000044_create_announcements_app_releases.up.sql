CREATE TABLE announcements (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL CHECK (BTRIM(title) <> ''),
    content TEXT NOT NULL CHECK (BTRIM(content) <> ''),
    type VARCHAR(32) NOT NULL CHECK (type IN ('normal', 'update', 'maintenance', 'warning', 'event', 'system')),
    display_mode VARCHAR(32) NOT NULL CHECK (display_mode IN ('normal', 'banner', 'modal', 'startup_modal')),
    target VARCHAR(32) NOT NULL CHECK (target IN ('all', 'web', 'android', 'ios', 'desktop')),
    priority INT NOT NULL DEFAULT 0,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    dismissible BOOLEAN NOT NULL DEFAULT TRUE,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_announcements_schedule CHECK (starts_at IS NULL OR ends_at IS NULL OR ends_at >= starts_at)
);

CREATE INDEX idx_announcements_active
    ON announcements (target, priority DESC, updated_at DESC)
    WHERE published = TRUE;
CREATE INDEX idx_announcements_created_by ON announcements (created_by) WHERE created_by IS NOT NULL;

CREATE TABLE app_releases (
    id BIGSERIAL PRIMARY KEY,
    platform VARCHAR(32) NOT NULL CHECK (platform IN ('android', 'ios', 'windows', 'macos', 'linux')),
    version_name VARCHAR(64) NOT NULL CHECK (BTRIM(version_name) <> ''),
    version_code BIGINT NOT NULL CHECK (version_code >= 0),
    title VARCHAR(255) NOT NULL CHECK (BTRIM(title) <> ''),
    changelog TEXT NOT NULL CHECK (BTRIM(changelog) <> ''),
    download_url VARCHAR(2048) NOT NULL CHECK (download_url ~ '^https?://'),
    file_size BIGINT CHECK (file_size >= 0),
    file_sha256 VARCHAR(64) CHECK (file_sha256 ~ '^[0-9a-f]{64}$'),
    minimum_version_code BIGINT NOT NULL DEFAULT 0,
    force_update BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'disabled')),
    published_at TIMESTAMPTZ,
    announcement_id BIGINT REFERENCES announcements(id) ON DELETE SET NULL,
    announcement_managed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_app_releases_platform_version_code UNIQUE (platform, version_code),
    CONSTRAINT chk_app_releases_minimum_version_code CHECK (minimum_version_code >= 0 AND minimum_version_code <= version_code),
    CONSTRAINT chk_app_releases_publication CHECK (status <> 'published' OR published_at IS NOT NULL)
);

CREATE INDEX idx_app_releases_latest_published
    ON app_releases (platform, version_code DESC)
    WHERE status = 'published';
CREATE UNIQUE INDEX uk_app_releases_announcement_id
    ON app_releases (announcement_id)
    WHERE announcement_id IS NOT NULL;
