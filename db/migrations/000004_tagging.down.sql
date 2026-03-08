-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_task_tag_tag;
DROP INDEX IF EXISTS idx_task_tag_task;
DROP INDEX IF EXISTS idx_tag_project;

DROP TABLE IF EXISTS task_tag;
DROP TABLE IF EXISTS tag;

-- +goose StatementEnd