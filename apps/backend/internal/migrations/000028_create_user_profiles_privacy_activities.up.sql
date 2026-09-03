CREATE TABLE user_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(100) NOT NULL,
    avatar_asset_id BIGINT REFERENCES image_assets(id) ON DELETE SET NULL,
    banner_asset_id BIGINT REFERENCES image_assets(id) ON DELETE SET NULL,
    bio VARCHAR(1000) NOT NULL DEFAULT '',
    gender VARCHAR(20) NOT NULL DEFAULT '',
    location VARCHAR(100) NOT NULL DEFAULT '',
    birthday DATE,
    website_url VARCHAR(2048) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_profiles_display_name_not_blank CHECK (BTRIM(display_name) <> '')
);

CREATE INDEX idx_user_profiles_avatar_asset_id ON user_profiles (avatar_asset_id);
CREATE INDEX idx_user_profiles_banner_asset_id ON user_profiles (banner_asset_id);

CREATE TABLE user_privacy_settings (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    profile_visibility VARCHAR(20) NOT NULL DEFAULT 'public',
    show_location BOOLEAN NOT NULL DEFAULT FALSE,
    show_birthday BOOLEAN NOT NULL DEFAULT FALSE,
    show_posts BOOLEAN NOT NULL DEFAULT TRUE,
    show_comments BOOLEAN NOT NULL DEFAULT TRUE,
    show_ratings BOOLEAN NOT NULL DEFAULT TRUE,
    show_favorites BOOLEAN NOT NULL DEFAULT FALSE,
    show_activity BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_privacy_profile_visibility_check
        CHECK (profile_visibility IN ('public', 'registered', 'private'))
);

INSERT INTO user_profiles (user_id, display_name, avatar_asset_id, created_at, updated_at)
SELECT id, username, avatar_asset_id, COALESCE(created_at, NOW()), NOW()
FROM users;

INSERT INTO user_privacy_settings (user_id)
SELECT id FROM users;

CREATE FUNCTION create_user_profile_defaults() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_profiles (user_id, display_name, avatar_asset_id)
    VALUES (NEW.id, NEW.username, NEW.avatar_asset_id);
    INSERT INTO user_privacy_settings (user_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_create_profile_defaults
AFTER INSERT ON users
FOR EACH ROW EXECUTE FUNCTION create_user_profile_defaults();

CREATE TABLE user_activities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    galgame_id BIGINT REFERENCES galgames(id) ON DELETE SET NULL,
    post_id BIGINT REFERENCES posts(id) ON DELETE SET NULL,
    comment_id BIGINT REFERENCES comments(id) ON DELETE SET NULL,
    resource_id BIGINT REFERENCES resources(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_activities_type_check CHECK (activity_type IN (
        'post_created', 'comment_created', 'rating_created',
        'favorite_created', 'resource_submitted', 'review_approved'
    )),
    CONSTRAINT user_activities_metadata_object_check CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT user_activities_single_entity_check CHECK (
        num_nonnulls(galgame_id, post_id, comment_id, resource_id) <= 1
    )
);

CREATE INDEX idx_user_activities_user_created_id
    ON user_activities (user_id, created_at DESC, id DESC);
CREATE INDEX idx_user_activities_galgame_id ON user_activities (galgame_id) WHERE galgame_id IS NOT NULL;
CREATE INDEX idx_user_activities_post_id ON user_activities (post_id) WHERE post_id IS NOT NULL;
CREATE INDEX idx_user_activities_comment_id ON user_activities (comment_id) WHERE comment_id IS NOT NULL;
CREATE INDEX idx_user_activities_resource_id ON user_activities (resource_id) WHERE resource_id IS NOT NULL;

CREATE INDEX idx_posts_author_created_id ON posts (author_id, created_at DESC, id DESC);
CREATE INDEX idx_comments_author_created_id ON comments (author_id, created_at DESC, id DESC);
CREATE INDEX idx_galgame_ratings_user_updated_id ON galgame_ratings (user_id, updated_at DESC, id DESC);
CREATE INDEX idx_galgame_favorites_user_created_id ON galgame_favorites (user_id, created_at DESC, id DESC);
