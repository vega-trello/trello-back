-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS sso_user CASCADE;
DROP TABLE IF EXISTS manual_user CASCADE;
DROP TABLE IF EXISTS base_user CASCADE;
DROP EXTENSION IF EXISTS "pgcrypto";
-- +goose StatementEnd