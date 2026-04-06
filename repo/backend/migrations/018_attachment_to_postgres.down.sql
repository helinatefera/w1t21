ALTER TABLE messages ADD COLUMN attachment_path TEXT;
ALTER TABLE messages ADD COLUMN attachment_size INT;
ALTER TABLE messages ADD COLUMN attachment_mime VARCHAR(100);

DROP TABLE IF EXISTS message_attachments;
