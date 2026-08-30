DROP TABLE IF EXISTS user_galgames;
DROP TABLE IF EXISTS galgame_favorites;
DROP TABLE IF EXISTS galgame_ratings;

ALTER TABLE galgames
    ALTER COLUMN rating_average TYPE NUMERIC(3,2);
