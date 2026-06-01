-- +goose Down
-- +goose StatementBegin

DELETE FROM role_permission WHERE role_id IN (1, 2, 3, 4);

DELETE FROM permission WHERE id <= 9;

DELETE FROM role WHERE id IN (1, 2, 3, 4);

DROP TABLE IF EXISTS project_member CASCADE;
DROP TABLE IF EXISTS role_permission CASCADE;
DROP TABLE IF EXISTS role CASCADE;
DROP TABLE IF EXISTS permission CASCADE;
DROP TABLE IF EXISTS project CASCADE;

-- +goose StatementEnd