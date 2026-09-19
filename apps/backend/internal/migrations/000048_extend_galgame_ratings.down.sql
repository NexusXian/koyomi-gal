DROP TABLE IF EXISTS galgame_rating_likes;

DROP INDEX IF EXISTS idx_galgame_ratings_game_popular;
DROP INDEX IF EXISTS idx_galgame_ratings_game_score;
DROP INDEX IF EXISTS idx_galgame_ratings_game_newest;

ALTER TABLE galgame_ratings
    DROP COLUMN IF EXISTS like_count,
    DROP COLUMN IF EXISTS spoiler_level,
    DROP COLUMN IF EXISTS review_text,
    DROP COLUMN IF EXISTS recommendation,
    DROP COLUMN IF EXISTS replay,
    DROP COLUMN IF EXISTS voice,
    DROP COLUMN IF EXISTS system,
    DROP COLUMN IF EXISTS branch,
    DROP COLUMN IF EXISTS character,
    DROP COLUMN IF EXISTS music,
    DROP COLUMN IF EXISTS story,
    DROP COLUMN IF EXISTS visual;
