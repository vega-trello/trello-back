-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS activity_log CASCADE;

-- +goose StatementEnd