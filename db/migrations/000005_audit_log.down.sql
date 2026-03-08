-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_activity_log_action;
DROP INDEX IF EXISTS idx_activity_log_entity;
DROP INDEX IF EXISTS idx_activity_log_created;
DROP INDEX IF EXISTS idx_activity_log_user;
DROP INDEX IF EXISTS idx_activity_log_project;

DROP TABLE IF EXISTS activity_log;

-- +goose StatementEnd