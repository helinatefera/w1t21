package model

import (
	"time"

	"github.com/google/uuid"
)

type Collectible struct {
	ID              uuid.UUID  `json:"id"`
	SellerID        uuid.UUID  `json:"seller_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	ContractAddress string     `json:"contract_address,omitempty"`
	ChainID         int        `json:"chain_id,omitempty"`
	TokenID         string     `json:"token_id,omitempty"`
	MetadataURI     string     `json:"metadata_uri,omitempty"`
	ImageURL        string     `json:"image_url,omitempty"`
	PriceCents      int64      `json:"price_cents"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	HiddenBy        *uuid.UUID `json:"hidden_by,omitempty"`
	HiddenReason    string     `json:"hidden_reason,omitempty"`
	ViewCount       int        `json:"view_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CollectibleTxHistory struct {
	ID             uuid.UUID `json:"id"`
	CollectibleID  uuid.UUID `json:"collectible_id"`
	TxHash         string    `json:"tx_hash"`
	FromAddress    string    `json:"from_address"`
	ToAddress      string    `json:"to_address"`
	BlockNumber    int64     `json:"block_number"`
	Timestamp      time.Time `json:"timestamp"`
}
