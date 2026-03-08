-- +goose Up
-- +goose StatementBegin

CREATE TABLE activity_log (
    id SERIAL PRIMARY KEY,
    project_uuid UUID REFERENCES project(uuid) ON DELETE SET NULL,
    user_uuid UUID REFERENCES base_user(uuid) ON DELETE SET NULL,
    action_type VARCHAR(32) NOT NULL,
    entity_type VARCHAR(16),
    entity_uuid UUID,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_activity_log_project ON activity_log(project_uuid);
CREATE INDEX idx_activity_log_user ON activity_log(user_uuid);
CREATE INDEX idx_activity_log_created ON activity_log(created_at DESC);
CREATE INDEX idx_activity_log_entity ON activity_log(entity_type, entity_uuid)
    WHERE entity_type IS NOT NULL;
CREATE INDEX idx_activity_log_action ON activity_log(action_type);

-- +goose StatementEnd