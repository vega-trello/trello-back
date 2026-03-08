-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_project_member_role;
DROP INDEX IF EXISTS idx_project_member_user;
DROP INDEX IF EXISTS idx_role_project;
DROP INDEX IF EXISTS idx_permission_name;
DROP INDEX IF EXISTS idx_project_title;

DROP TABLE IF EXISTS project_member;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS role;
DROP TABLE IF EXISTS permission;
DROP TABLE IF EXISTS project;

-- +goose StatementEnd