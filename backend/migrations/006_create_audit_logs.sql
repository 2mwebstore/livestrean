-- Security and admin events. No foreign key on user_id so entries survive
-- when a user is deleted.
CREATE TABLE audit_logs (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NULL,
  action VARCHAR(50) NOT NULL,
  resource VARCHAR(50) NOT NULL DEFAULT '',
  resource_id VARCHAR(64) NOT NULL DEFAULT '',
  ip VARCHAR(45) NOT NULL DEFAULT '',
  user_agent VARCHAR(255) NOT NULL DEFAULT '',
  success TINYINT(1) NOT NULL,
  created_at DATETIME(3) NOT NULL,
  KEY idx_audit_logs_user (user_id),
  KEY idx_audit_logs_action (action, created_at)
);
