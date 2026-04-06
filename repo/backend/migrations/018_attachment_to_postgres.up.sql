-- Move attachment storage from filesystem to PostgreSQL (system of record).
-- A separate table keeps large BYTEA blobs out of message listing queries.

CREATE TABLE message_attachments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL UNIQUE REFERENCES messages(id) ON DELETE CASCADE,
    data       BYTEA NOT NULL,
    size       INT NOT NULL,
    mime       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_message_attachments_message ON message_attachments(message_id);

-- Drop the filesystem-only columns from messages — attachment metadata now
-- lives in the message_attachments table. Keep lightweight pointers so the
-- listing query can still report whether an attachment exists.
ALTER TABLE messages DROP COLUMN attachment_path;
ALTER TABLE messages DROP COLUMN attachment_size;
ALTER TABLE messages DROP COLUMN attachment_mime;
