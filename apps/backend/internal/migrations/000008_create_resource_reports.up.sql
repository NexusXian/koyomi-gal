CREATE TABLE resource_reports (
    id BIGSERIAL PRIMARY KEY,
    resource_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    reason SMALLINT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status SMALLINT NOT NULL DEFAULT 0,
    handled_by BIGINT,
    handled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_resource_reports_resource
        FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE,
    CONSTRAINT fk_resource_reports_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_resource_reports_handled_by
        FOREIGN KEY (handled_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT resource_reports_reason_range CHECK (reason >= 0 AND reason <= 6),
    CONSTRAINT resource_reports_status_range CHECK (status >= 0 AND status <= 2)
);

CREATE UNIQUE INDEX resource_reports_unique ON resource_reports (resource_id, user_id);
CREATE INDEX idx_resource_reports_status ON resource_reports (status);
CREATE INDEX idx_resource_reports_user_id ON resource_reports (user_id);
