CREATE TABLE level_configs (
    id BIGSERIAL PRIMARY KEY,
    level INT NOT NULL,
    name VARCHAR(50) NOT NULL,
    min_exp BIGINT NOT NULL DEFAULT 0,
    icon_url VARCHAR(2048) NOT NULL DEFAULT '',
    color VARCHAR(32) NOT NULL DEFAULT '',
    description VARCHAR(255) NOT NULL DEFAULT '',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT level_configs_level_unique UNIQUE (level),
    CONSTRAINT level_configs_min_exp_check CHECK (min_exp >= 0)
);

INSERT INTO level_configs (level, name, min_exp, description) VALUES
    (1, '初见', 0, '刚刚踏入 Koyomi Gal 的世界'),
    (2, '读者', 100, '开始浏览游戏资料与社区内容'),
    (3, '爱好者', 500, '积极参与社区互动'),
    (4, '鉴赏家', 1500, '对 Galgame 有自己的见解'),
    (5, '资深鉴赏家', 4000, '持续贡献内容与资源'),
    (6, '收藏家', 10000, '收藏了大量珍贵资源'),
    (7, '研究者', 25000, '深入研究游戏资料与背景'),
    (8, '大师', 60000, '社区公认的 Galgame 大师');

CREATE TABLE user_experience (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_exp BIGINT NOT NULL DEFAULT 0,
    current_level INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_experience_total_exp_check CHECK (total_exp >= 0)
);

CREATE TABLE experience_rules (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    name VARCHAR(100) NOT NULL,
    exp INT NOT NULL,
    daily_limit INT NOT NULL DEFAULT 0,
    daily_exp_limit INT NOT NULL DEFAULT 0,
    cooldown_seconds INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT experience_rules_event_type_unique UNIQUE (event_type),
    CONSTRAINT experience_rules_exp_check CHECK (exp >= 0),
    CONSTRAINT experience_rules_daily_limit_check CHECK (daily_limit >= 0),
    CONSTRAINT experience_rules_daily_exp_limit_check CHECK (daily_exp_limit >= 0),
    CONSTRAINT experience_rules_cooldown_check CHECK (cooldown_seconds >= 0)
);

INSERT INTO experience_rules (event_type, name, exp, daily_limit, daily_exp_limit, cooldown_seconds, description) VALUES
    ('daily_checkin', '每日签到', 10, 1, 0, 0, '每天首次签到获得经验'),
    ('comment_created', '发布评论', 2, 5, 0, 0, '发布评论获得经验，每日最多 5 次'),
    ('like_given', '点赞内容', 1, 5, 0, 0, '主动点赞内容获得经验，每日最多 5 次'),
    ('comment_liked', '评论被点赞', 2, 0, 20, 0, '评论被他人点赞获得经验，每日上限 20'),
    ('share', '分享内容', 3, 3, 0, 300, '分享内容获得经验，每日最多 3 次'),
    ('game_contribution_approved', '游戏资料贡献审核通过', 30, 0, 0, 0, '提交的游戏资料通过审核'),
    ('resource_approved', '资源贡献审核通过', 50, 0, 0, 0, '提交的资源通过审核'),
    ('cg_approved', 'CG 贡献审核通过', 20, 0, 0, 0, '提交的游戏画面通过审核'),
    ('character_approved', '角色资料贡献审核通过', 20, 0, 0, 0, '提交的角色资料通过审核'),
    ('article_approved', '文章审核通过', 40, 0, 0, 0, '提交的文章通过审核'),
    ('admin_adjustment', '管理员调整', 0, 0, 0, 0, '管理员手动调整经验的记录事件，经验值以手动输入为准');

CREATE TABLE experience_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    exp_delta BIGINT NOT NULL,
    source_type VARCHAR(32) NOT NULL DEFAULT '',
    source_id BIGINT,
    idempotency_key VARCHAR(128),
    description VARCHAR(255) NOT NULL DEFAULT '',
    operator_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT experience_logs_idempotency_key_unique UNIQUE (idempotency_key)
);

CREATE INDEX idx_experience_logs_user_created
    ON experience_logs (user_id, created_at DESC, id DESC);

CREATE INDEX idx_experience_logs_event_type
    ON experience_logs (event_type);

CREATE TABLE user_checkins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date DATE NOT NULL,
    consecutive_days INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_checkins_user_date_unique UNIQUE (user_id, checkin_date)
);

CREATE INDEX idx_user_checkins_user_date
    ON user_checkins (user_id, checkin_date DESC);
