ALTER TABLE galgames
    ALTER COLUMN rating_average TYPE NUMERIC(4,2);

CREATE TABLE galgame_ratings (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    score SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_galgame_ratings_galgame
        FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE,
    CONSTRAINT fk_galgame_ratings_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT galgame_ratings_score_range CHECK (score >= 1 AND score <= 10)
);

CREATE UNIQUE INDEX galgame_ratings_unique ON galgame_ratings (galgame_id, user_id);
CREATE INDEX idx_galgame_ratings_user_id ON galgame_ratings (user_id);

CREATE TABLE galgame_favorites (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_galgame_favorites_galgame
        FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE,
    CONSTRAINT fk_galgame_favorites_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX galgame_favorites_unique ON galgame_favorites (galgame_id, user_id);
CREATE INDEX idx_galgame_favorites_user_id ON galgame_favorites (user_id);

CREATE TABLE user_galgames (
    id BIGSERIAL PRIMARY KEY,
    galgame_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    state SMALLINT NOT NULL,
    play_time_minutes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user_galgames_galgame
        FOREIGN KEY (galgame_id) REFERENCES galgames(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_galgames_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT user_galgames_state_range CHECK (state >= 1 AND state <= 5),
    CONSTRAINT user_galgames_play_time_nonnegative CHECK (play_time_minutes >= 0)
);

CREATE UNIQUE INDEX user_galgames_unique ON user_galgames (galgame_id, user_id);
CREATE INDEX idx_user_galgames_user_id ON user_galgames (user_id);
