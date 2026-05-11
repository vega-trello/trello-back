-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS task_assignee CASCADE;
DROP TABLE IF EXISTS tasks CASCADE;
DROP TABLE IF EXISTS project_status CASCADE;
DROP TABLE IF EXISTS project_column CASCADE;

-- +goose StatementEnd