CREATE TABLE feedback (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(32) NOT NULL CHECK (type IN ('feedback', 'copyright')),
    content TEXT NOT NULL,
    contact VARCHAR(255) NOT NULL DEFAULT '',
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent VARCHAR(512) NOT NULL DEFAULT '',
    handled_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    handled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feedback_admin_order ON feedback (id DESC);
CREATE INDEX idx_feedback_type_handled ON feedback (type, handled_at) WHERE handled_at IS NULL;
