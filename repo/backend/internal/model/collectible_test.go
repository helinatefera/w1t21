package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCollectibleTxHistory_Fields(t *testing.T) {
	id := uuid.New()
	cid := uuid.New()
	now := time.Now()

	h := CollectibleTxHistory{
		ID:            id,
		CollectibleID: cid,
		TxHash:        "order:abc-123",
		FromAddress:   "0xSeller",
		ToAddress:     "0xBuyer",
		BlockNumber:   99900,
		Timestamp:     now,
	}

	if h.ID != id {
		t.Errorf("expected ID %v, got %v", id, h.ID)
	}
	if h.TxHash != "order:abc-123" {
		t.Errorf("expected TxHash order:abc-123, got %s", h.TxHash)
	}
	if h.FromAddress != "0xSeller" {
		t.Errorf("expected FromAddress 0xSeller, got %s", h.FromAddress)
	}
	if h.ToAddress != "0xBuyer" {
		t.Errorf("expected ToAddress 0xBuyer, got %s", h.ToAddress)
	}
	if h.BlockNumber != 99900 {
		t.Errorf("expected BlockNumber 99900, got %d", h.BlockNumber)
	}
}
