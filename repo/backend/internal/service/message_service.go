package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

const MaxAttachmentSize = 10 * 1024 * 1024 // 10 MB

type MessageService struct {
	messageStore *store.MessageStore
	orderStore   *store.OrderStore
}

func NewMessageService(ms *store.MessageStore, os *store.OrderStore) *MessageService {
	return &MessageService{messageStore: ms, orderStore: os}
}

func (s *MessageService) GetByID(ctx context.Context, msgID, userID uuid.UUID) (*model.Message, error) {
	msg, err := s.messageStore.GetByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, dto.ErrNotFound
	}
	// Verify user is participant on the order
	order, err := s.orderStore.GetByID(ctx, msg.OrderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, dto.ErrNotFound
	}
	if order.BuyerID != userID && order.SellerID != userID {
		return nil, dto.ErrForbidden
	}
	return msg, nil
}

// GetAttachment returns the full attachment binary for a message after
// verifying the caller is a participant on the parent order.
func (s *MessageService) GetAttachment(ctx context.Context, msgID, userID uuid.UUID) (*model.MessageAttachment, error) {
	msg, err := s.messageStore.GetByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, dto.ErrNotFound
	}

	order, err := s.orderStore.GetByID(ctx, msg.OrderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, dto.ErrNotFound
	}
	if order.BuyerID != userID && order.SellerID != userID {
		return nil, dto.ErrForbidden
	}

	att, err := s.messageStore.GetAttachmentByMessageID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if att == nil {
		return nil, dto.ErrNotFound
	}
	return att, nil
}

func (s *MessageService) Send(ctx context.Context, orderID uuid.UUID, senderID uuid.UUID, body string,
	attachmentData []byte, attachmentSize int, attachmentMime string) (*model.Message, error) {

	// Verify order exists and sender is buyer or seller
	order, err := s.orderStore.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, dto.ErrNotFound
	}
	if order.BuyerID != senderID && order.SellerID != senderID {
		return nil, dto.ErrForbidden
	}

	// Server-side attachment size check
	if attachmentSize > MaxAttachmentSize {
		return nil, dto.ErrAttachmentTooLarge
	}

	msg := &model.Message{
		OrderID:  orderID,
		SenderID: senderID,
		Body:     body,
	}

	if err := s.messageStore.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Persist attachment in the message_attachments table (PostgreSQL as sole system of record).
	if len(attachmentData) > 0 {
		att := &model.MessageAttachment{
			MessageID: msg.ID,
			Data:      attachmentData,
			Size:      attachmentSize,
			Mime:      attachmentMime,
		}
		if err := s.messageStore.CreateAttachment(ctx, att); err != nil {
			return nil, fmt.Errorf("create attachment: %w", err)
		}
		msg.AttachmentID = msg.ID.String()
		msg.AttachmentSize = attachmentSize
		msg.AttachmentMime = attachmentMime
	}

	return msg, nil
}

func (s *MessageService) ListByOrder(ctx context.Context, orderID, userID uuid.UUID, page, pageSize int) ([]model.Message, int, error) {
	// Verify user is participant
	order, err := s.orderStore.GetByID(ctx, orderID)
	if err != nil {
		return nil, 0, err
	}
	if order == nil {
		return nil, 0, dto.ErrNotFound
	}
	if order.BuyerID != userID && order.SellerID != userID {
		return nil, 0, dto.ErrForbidden
	}

	return s.messageStore.ListByOrder(ctx, orderID, page, pageSize)
}
