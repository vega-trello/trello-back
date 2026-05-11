-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS project_member CASCADE;
DROP TABLE IF EXISTS role_permission CASCADE;
DROP TABLE IF EXISTS role CASCADE;
DROP TABLE IF EXISTS permission CASCADE;
DROP TABLE IF EXISTS project CASCADE;

-- +goose StatementEnd