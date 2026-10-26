-- Next scheduled stream (for the countdown) and the channel thumbnail.
ALTER TABLE channels
  ADD COLUMN scheduled_at DATETIME(3) NULL AFTER last_live_at,
  ADD COLUMN thumbnail_version VARCHAR(16) NULL AFTER scheduled_at;

-- Thumbnails live in the database so they survive redeploys on platforms
-- without persistent disks. Always a re-encoded 1280x720 (or smaller) JPEG.
CREATE TABLE channel_thumbnails (
  channel_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  content_type VARCHAR(50) NOT NULL,
  data MEDIUMBLOB NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  CONSTRAINT fk_channel_thumbnails_channel FOREIGN KEY (channel_id) REFERENCES channels (id) ON DELETE CASCADE
);
