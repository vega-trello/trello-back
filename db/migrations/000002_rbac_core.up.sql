-- +goose Up
-- +goose StatementBegin

CREATE TABLE project (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(128) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW() -- в гошке обновлять
);

CREATE INDEX idx_project_title ON project(title);

CREATE TABLE permission (
    id SERIAL PRIMARY KEY,
    name VARCHAR(32) NOT NULL UNIQUE,
    description VARCHAR(256)
);

CREATE INDEX idx_permission_name ON permission(name);

CREATE TABLE role (
    id SERIAL PRIMARY KEY,
    project_uuid UUID REFERENCES project(uuid) ON DELETE CASCADE,
    name VARCHAR(32) NOT NULL,
    description VARCHAR(256),
    UNIQUE(project_uuid, name)
);

CREATE TABLE role_permission (
    role_id INTEGER NOT NULL REFERENCES role(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permission(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permission_role ON role_permission(role_id);
CREATE INDEX idx_role_permission_permission ON role_permission(permission_id);

CREATE UNIQUE INDEX idx_role_global_name_unique
    ON role (name)
    WHERE project_uuid IS NULL;

CREATE INDEX idx_role_project ON role(project_uuid);

CREATE TABLE project_member (
    project_uuid UUID NOT NULL REFERENCES project(uuid) ON DELETE CASCADE,
    user_uuid UUID NOT NULL REFERENCES base_user(uuid) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES role(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_uuid, user_uuid)
);

CREATE INDEX idx_project_member_user ON project_member(user_uuid);
CREATE INDEX idx_project_member_role ON project_member(role_id);

INSERT INTO role (id, name, description, project_uuid) VALUES
    (1, 'owner', 'Owner', NULL),                                                       (2, 'admin', 'Admin', NULL),
    (3, 'member', 'Member', NULL),
    (4, 'viewer', 'Viewer', NULL)
ON CONFLICT (id) DO NOTHING;

SELECT setval('role_id_seq', (SELECT MAX(id) FROM role));
-- +goose StatementEnd