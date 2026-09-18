DROP TABLE IF EXISTS user_ip_logs;

ALTER TABLE comments
    DROP COLUMN IF EXISTS ip_region,
    DROP COLUMN IF EXISTS ip_address;

ALTER TABLE posts
    DROP COLUMN IF EXISTS ip_region,
    DROP COLUMN IF EXISTS ip_address;
