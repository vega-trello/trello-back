-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS task_tag CASCADE;
DROP TABLE IF EXISTS tag CASCADE;
-- +goose StatementEnd