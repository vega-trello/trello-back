-- +goose Down
-- +goose StatementBegin

ALTER TABLE project
    DROP COLUMN description;

-- +goose StatementEnd