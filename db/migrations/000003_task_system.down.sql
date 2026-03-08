-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_task_assignee_user;
DROP INDEX IF EXISTS idx_tasks_archived;
DROP INDEX IF EXISTS idx_tasks_deleted;
DROP INDEX IF EXISTS idx_tasks_creator;
DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_column;

DROP TABLE IF EXISTS task_assignee;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS project_status;
DROP TABLE IF EXISTS project_column;

-- +goose StatementEnd