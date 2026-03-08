-- +goose Up
-- +goose StatementBegin

CREATE TABLE project_column (
    id SERIAL PRIMARY KEY,
    project_uuid UUID NOT NULL REFERENCES project(uuid) ON DELETE CASCADE,
    name VARCHAR(64) NOT NULL,
    position INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_uuid, position)
);

CREATE INDEX idx_project_column_project ON project_column(project_uuid);

CREATE TABLE project_status (
    id SERIAL PRIMARY KEY,
    project_uuid UUID NOT NULL REFERENCES project(uuid) ON DELETE CASCADE,
    name VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_uuid, name)
);

CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    column_id INTEGER REFERENCES project_column(id) ON DELETE SET NULL,
    status_id INTEGER REFERENCES project_status(id) ON DELETE SET NULL,
    creator_uuid UUID NOT NULL REFERENCES base_user(uuid) ON DELETE CASCADE,
    title TEXT,
    description TEXT,
    position INTEGER NOT NULL DEFAULT 0,
    deleted_at TIMESTAMPTZ, -- обновлять в гошке
    archived_at TIMESTAMPTZ, --обновлять в гошке
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    UNIQUE (column_id, position)
);

CREATE INDEX idx_tasks_column ON tasks(column_id);
CREATE INDEX idx_tasks_status ON tasks(status_id);
CREATE INDEX idx_tasks_creator ON tasks(creator_uuid);
CREATE INDEX idx_tasks_deleted ON tasks(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_tasks_archived ON tasks(archived_at) WHERE archived_at IS NOT NULL;


CREATE TABLE task_assignee (                              task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_uuid UUID NOT NULL REFERENCES base_user(uuid) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, user_uuid)
);

CREATE INDEX idx_task_assignee_user ON task_assignee(user_uuid);

-- +goose StatementEnd