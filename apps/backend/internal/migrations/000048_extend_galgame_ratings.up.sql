ALTER TABLE galgame_ratings
    ADD COLUMN visual SMALLINT,
    ADD COLUMN story SMALLINT,
    ADD COLUMN music SMALLINT,
    ADD COLUMN character SMALLINT,
    ADD COLUMN branch SMALLINT,
    ADD COLUMN system SMALLINT,
    ADD COLUMN voice SMALLINT,
    ADD COLUMN replay SMALLINT,
    ADD COLUMN recommendation SMALLINT,
    ADD COLUMN review_text TEXT,
    ADD COLUMN spoiler_level SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN like_count BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT galgame_ratings_visual_range CHECK (visual BETWEEN 1 AND 10),
    ADD CONSTRAINT galgame_ratings_story_range CHECK (story BETWEEN 1 AND 10),
    ADD CONSTRAINT galgame_ratings_music_range CHECK (music BETWEEN 1 AND 10),
    ADD CONSTRAINT galgame_ratings_character_range CHECK (character BETWEEN 1 AND 10),
    ADD CONSTRAINT galgame_ratings_branch_range CHECK (branch BETWEEN 1 AND 10),
    ADD CONSTRAINT galgame_ratings_system_range CHECK (system BETWEEN 1 AND 10),
    ADD CONSTRAINT galgame_ratings_voice_range CHECK (voice BETWEEN 1 AND 10),
    ADD CONSTRAINT galgame_ratings_replay_range CHECK (replay BETWEEN 1 AND 10),
    ADD CONSTRAINT galgame_ratings_recommendation_range CHECK (recommendation IN (-1, 0, 1, 2)),
    ADD CONSTRAINT galgame_ratings_spoiler_level_range CHECK (spoiler_level BETWEEN 0 AND 2),
    ADD CONSTRAINT galgame_ratings_like_count_nonnegative CHECK (like_count >= 0);

CREATE INDEX idx_galgame_ratings_game_newest
    ON galgame_ratings (galgame_id, updated_at DESC, id DESC);
CREATE INDEX idx_galgame_ratings_game_score
    ON galgame_ratings (galgame_id, score DESC, id DESC);
CREATE INDEX idx_galgame_ratings_game_popular
    ON galgame_ratings (galgame_id, like_count DESC, updated_at DESC, id DESC);

CREATE TABLE galgame_rating_likes (
    id BIGSERIAL PRIMARY KEY,
    rating_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_galgame_rating_likes_rating
        FOREIGN KEY (rating_id) REFERENCES galgame_ratings(id) ON DELETE CASCADE,
    CONSTRAINT fk_galgame_rating_likes_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT galgame_rating_likes_unique UNIQUE (rating_id, user_id)
);

CREATE INDEX idx_galgame_rating_likes_user_id ON galgame_rating_likes (user_id);
