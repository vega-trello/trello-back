-- +goose Up
-- +goose StatementBegin

CREATE TABLE project (
     uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
     title VARCHAR(128) NOT NULL,
     description TEXT,
     created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
     updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
CREATE UNIQUE INDEX idx_role_global_name_unique ON role (name) WHERE project_uuid IS NULL;
CREATE INDEX idx_role_project ON role(project_uuid);

CREATE TABLE project_member (
    project_uuid UUID NOT NULL REFERENCES project(uuid) ON DELETE CASCADE,
    user_uuid    UUID NOT NULL REFERENCES base_user(uuid) ON DELETE CASCADE,
    role_id      INTEGER NOT NULL REFERENCES role(id) ON DELETE CASCADE,
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_uuid, user_uuid)
);

CREATE INDEX idx_project_member_user ON project_member(user_uuid);
CREATE INDEX idx_project_member_role ON project_member(role_id);

INSERT INTO role (id, name, description, project_uuid) VALUES
   (1, 'Создатель', 'Создатель проекта, обладающий всеми разрешениями', NULL),
   (2, 'Администратор', 'Управление проектом и участниками, без удаления проекта', NULL),
   (3, 'Участник', 'Базовый доступ: просмотр и работа с задачами', NULL),
   (4, 'Наблюдатель', 'Только просмотр проекта без права редактирования', NULL)
ON CONFLICT (id) DO NOTHING;

SELECT setval('role_id_seq', (SELECT MAX(id) FROM role));

INSERT INTO permission (id, name, description) VALUES
   (1, 'view_project', 'Просматривать проект, его задачи и участников'),
   (2, 'manage_project', 'Редактировать настройки проекта и управлять его конфигурацией'),
   (3, 'manage_members', 'Добавлять и удалять участников проекта'),
   (4, 'manage_roles', 'Создавать, изменять и назначать роли участникам'),
   (5, 'manage_columns', 'Создавать и редактировать колонки доски'),
   (6, 'manage_tasks', 'Создавать, редактировать и удалять задачи'),
   (7, 'manage_statuses', 'Добавлять и изменять статусы задач'),
   (8, 'manage_tags', 'Создавать и редактировать теги для задач'),
   (9, 'manage_assignees', 'Назначать и снимать исполнителей с задач')
ON CONFLICT (id) DO NOTHING;

SELECT setval('permission_id_seq', (SELECT MAX(id) FROM permission));

INSERT INTO role_permission (role_id, permission_id)
SELECT 1, id FROM permission
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id)
SELECT 2, id FROM permission WHERE id != 2
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id) VALUES
     (3, 1), -- view_project
     (3, 5), -- manage_columns
     (3, 6), -- manage_tasks
     (3, 7), -- manage_statuses
     (3, 8), -- manage_tags
     (3, 9)  -- manage_assignees
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permission (role_id, permission_id) VALUES
    (4, 1)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- +goose StatementEnd