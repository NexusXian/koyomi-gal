ALTER TABLE posts
    ADD COLUMN ip_address VARCHAR(45) NOT NULL DEFAULT '',
    ADD COLUMN ip_region VARCHAR(100) NOT NULL DEFAULT '';

ALTER TABLE comments
    ADD COLUMN ip_address VARCHAR(45) NOT NULL DEFAULT '',
    ADD COLUMN ip_region VARCHAR(100) NOT NULL DEFAULT '';

CREATE TABLE user_ip_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    country VARCHAR(64) NOT NULL DEFAULT '',
    region VARCHAR(64) NOT NULL DEFAULT '',
    city VARCHAR(64) NOT NULL DEFAULT '',
    isp VARCHAR(128) NOT NULL DEFAULT '',
    action VARCHAR(32) NOT NULL,
    entity_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user_ip_logs_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_ip_logs_user_id_created_at
    ON user_ip_logs (user_id, created_at DESC);
CREATE INDEX idx_user_ip_logs_ip_address
    ON user_ip_logs (ip_address);
CREATE UNIQUE INDEX user_ip_logs_action_entity_unique
    ON user_ip_logs (action, entity_id)
    WHERE entity_id IS NOT NULL;
