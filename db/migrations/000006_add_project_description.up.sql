-- +goose Up
-- +goose StatementBegin

ALTER TABLE project
    ADD COLUMN description TEXT;

-- +goose StatementEnd