-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS sso_user;
DROP TABLE IF EXISTS manual_user;
DROP TABLE IF EXISTS base_user;

-- +goose StatementEnd