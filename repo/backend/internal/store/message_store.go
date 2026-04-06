package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledgermint/platform/internal/model"
)

type MessageStore struct {
	pool *pgxpool.Pool
}

func NewMessageStore(pool *pgxpool.Pool) *MessageStore {
	return &MessageStore{pool: pool}
}

func (s *MessageStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error) {
	var m model.Message
	err := s.pool.QueryRow(ctx,
		`SELECT m.id, m.order_id, m.sender_id, m.body, m.created_at,
		        COALESCE(a.size, 0), COALESCE(a.mime, '')
		 FROM messages m
		 LEFT JOIN message_attachments a ON a.message_id = m.id
		 WHERE m.id = $1`, id).Scan(
		&m.ID, &m.OrderID, &m.SenderID, &m.Body, &m.CreatedAt,
		&m.AttachmentSize, &m.AttachmentMime)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if m.AttachmentSize > 0 {
		m.AttachmentID = m.ID.String()
	}
	return &m, err
}

func (s *MessageStore) Create(ctx context.Context, m *model.Message) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO messages (order_id, sender_id, body)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		m.OrderID, m.SenderID, m.Body,
	).Scan(&m.ID, &m.CreatedAt)
}

// CreateAttachment inserts a row into message_attachments, storing the
// binary content in PostgreSQL. The size is validated by the caller.
func (s *MessageStore) CreateAttachment(ctx context.Context, a *model.MessageAttachment) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO message_attachments (message_id, data, size, mime)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		a.MessageID, a.Data, a.Size, a.Mime,
	).Scan(&a.ID, &a.CreatedAt)
}

// GetAttachmentByMessageID returns the full attachment (including binary data)
// for the given message. Returns nil if no attachment exists.
func (s *MessageStore) GetAttachmentByMessageID(ctx context.Context, messageID uuid.UUID) (*model.MessageAttachment, error) {
	var a model.MessageAttachment
	err := s.pool.QueryRow(ctx,
		`SELECT id, message_id, data, size, mime, created_at
		 FROM message_attachments WHERE message_id = $1`, messageID).Scan(
		&a.ID, &a.MessageID, &a.Data, &a.Size, &a.Mime, &a.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func (s *MessageStore) ListByOrder(ctx context.Context, orderID uuid.UUID, page, pageSize int) ([]model.Message, int, error) {
	var total int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM messages WHERE order_id = $1`, orderID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := s.pool.Query(ctx,
		`SELECT m.id, m.order_id, m.sender_id, m.body, m.created_at,
		        COALESCE(a.size, 0), COALESCE(a.mime, '')
		 FROM messages m
		 LEFT JOIN message_attachments a ON a.message_id = m.id
		 WHERE m.order_id = $1
		 ORDER BY m.created_at ASC LIMIT $2 OFFSET $3`, orderID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.OrderID, &m.SenderID, &m.Body, &m.CreatedAt,
			&m.AttachmentSize, &m.AttachmentMime); err != nil {
			return nil, 0, err
		}
		if m.AttachmentSize > 0 {
			m.AttachmentID = m.ID.String()
		}
		messages = append(messages, m)
	}
	return messages, total, nil
}
