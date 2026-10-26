-- A channel is one stream: its URL (slug), details and stream key.
-- Only a SHA-256 hash of the stream key is stored.
CREATE TABLE channels (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id BIGINT UNSIGNED NOT NULL,
  slug VARCHAR(40) NOT NULL UNIQUE,
  title VARCHAR(100) NOT NULL,
  description VARCHAR(1000) NOT NULL DEFAULT '',
  stream_key_hash CHAR(64) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  last_live_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  KEY idx_channels_owner (owner_id),
  KEY idx_channels_status_last_live (status, last_live_at),
  CONSTRAINT fk_channels_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE
);
