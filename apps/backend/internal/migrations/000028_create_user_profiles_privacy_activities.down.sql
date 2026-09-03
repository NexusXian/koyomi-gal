DROP INDEX IF EXISTS idx_galgame_favorites_user_created_id;
DROP INDEX IF EXISTS idx_galgame_ratings_user_updated_id;
DROP INDEX IF EXISTS idx_comments_author_created_id;
DROP INDEX IF EXISTS idx_posts_author_created_id;
DROP TABLE IF EXISTS user_activities;
DROP TRIGGER IF EXISTS users_create_profile_defaults ON users;
DROP FUNCTION IF EXISTS create_user_profile_defaults();
DROP TABLE IF EXISTS user_privacy_settings;
DROP TABLE IF EXISTS user_profiles;
