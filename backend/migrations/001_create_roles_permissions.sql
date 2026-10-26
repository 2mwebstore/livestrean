CREATE TABLE roles (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  slug VARCHAR(50) NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL
);

CREATE TABLE permissions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  slug VARCHAR(100) NOT NULL UNIQUE,
  description VARCHAR(255) NOT NULL
);

CREATE TABLE role_permissions (
  role_id BIGINT UNSIGNED NOT NULL,
  permission_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (role_id, permission_id),
  CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
  CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

INSERT INTO roles (slug, name) VALUES
  ('admin', 'Admin'),
  ('streamer', 'Streamer'),
  ('viewer', 'Viewer');

INSERT INTO permissions (slug, description) VALUES
  ('comments.create', 'Post chat comments'),
  ('comments.moderate', 'Delete any chat comment'),
  ('channels.create', 'Create own channels'),
  ('channels.manage_own', 'View, edit and delete own channels'),
  ('channels.view_all', 'View all channels in admin'),
  ('channels.update_all', 'Enable or disable any channel'),
  ('users.view', 'View users in admin'),
  ('users.update', 'Change a user''s role and status');

INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
  WHERE r.slug = 'admin';

INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
  WHERE r.slug = 'streamer'
    AND p.slug IN ('comments.create', 'channels.create', 'channels.manage_own');

INSERT INTO role_permissions (role_id, permission_id)
  SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
  WHERE r.slug = 'viewer'
    AND p.slug IN ('comments.create');
