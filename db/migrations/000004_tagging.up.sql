-- +goose Up
-- +goose StatementBegin

CREATE TABLE tag (
    id SERIAL PRIMARY KEY,
    project_uuid UUID NOT NULL REFERENCES project(uuid) ON DELETE CASCADE,
    name VARCHAR(64) NOT NULL,
    color INTEGER NOT NULL,  --парсить в го
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_uuid, name)
);

CREATE INDEX idx_tag_project ON tag(project_uuid);

CREATE TABLE task_tag (
    task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, tag_id)
);

CREATE INDEX idx_task_tag_task ON task_tag(task_id);
CREATE INDEX idx_task_tag_tag ON task_tag(tag_id);

-- +goose StatementEnd