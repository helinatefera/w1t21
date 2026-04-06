package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledgermint/platform/internal/model"
)

type CollectibleStore struct {
	pool *pgxpool.Pool
}

func NewCollectibleStore(pool *pgxpool.Pool) *CollectibleStore {
	return &CollectibleStore{pool: pool}
}

func (s *CollectibleStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Collectible, error) {
	var c model.Collectible
	err := s.pool.QueryRow(ctx,
		`SELECT id, seller_id, title, description,
		        COALESCE(contract_address, ''), COALESCE(chain_id, 0), COALESCE(token_id, ''),
		        COALESCE(metadata_uri, ''), COALESCE(image_url, ''), price_cents, currency, status, hidden_by,
		        COALESCE(hidden_reason, ''), view_count, created_at, updated_at
		 FROM collectibles WHERE id = $1`, id).Scan(
		&c.ID, &c.SellerID, &c.Title, &c.Description, &c.ContractAddress, &c.ChainID,
		&c.TokenID, &c.MetadataURI, &c.ImageURL, &c.PriceCents, &c.Currency, &c.Status,
		&c.HiddenBy, &c.HiddenReason, &c.ViewCount, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (s *CollectibleStore) List(ctx context.Context, status string, page, pageSize int) ([]model.Collectible, int, error) {
	var total int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM collectibles WHERE ($1 = '' OR status = $1)`, status).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := s.pool.Query(ctx,
		`SELECT id, seller_id, title, description,
		        COALESCE(contract_address, ''), COALESCE(chain_id, 0), COALESCE(token_id, ''),
		        COALESCE(metadata_uri, ''), COALESCE(image_url, ''), price_cents, currency, status, view_count, created_at, updated_at
		 FROM collectibles
		 WHERE ($1 = '' OR status = $1)
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var collectibles []model.Collectible
	for rows.Next() {
		var c model.Collectible
		if err := rows.Scan(&c.ID, &c.SellerID, &c.Title, &c.Description, &c.ContractAddress,
			&c.ChainID, &c.TokenID, &c.MetadataURI, &c.ImageURL, &c.PriceCents, &c.Currency,
			&c.Status, &c.ViewCount, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		collectibles = append(collectibles, c)
	}
	return collectibles, total, nil
}

func (s *CollectibleStore) ListBySeller(ctx context.Context, sellerID uuid.UUID, page, pageSize int) ([]model.Collectible, int, error) {
	var total int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM collectibles WHERE seller_id = $1`, sellerID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := s.pool.Query(ctx,
		`SELECT id, seller_id, title, description,
		        COALESCE(contract_address, ''), COALESCE(chain_id, 0), COALESCE(token_id, ''),
		        COALESCE(metadata_uri, ''), COALESCE(image_url, ''), price_cents, currency, status, view_count, created_at, updated_at
		 FROM collectibles WHERE seller_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, sellerID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var collectibles []model.Collectible
	for rows.Next() {
		var c model.Collectible
		if err := rows.Scan(&c.ID, &c.SellerID, &c.Title, &c.Description, &c.ContractAddress,
			&c.ChainID, &c.TokenID, &c.MetadataURI, &c.ImageURL, &c.PriceCents, &c.Currency,
			&c.Status, &c.ViewCount, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		collectibles = append(collectibles, c)
	}
	return collectibles, total, nil
}

func (s *CollectibleStore) Create(ctx context.Context, c *model.Collectible) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO collectibles (seller_id, title, description, contract_address, chain_id,
		        token_id, metadata_uri, image_url, price_cents, currency)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, status, view_count, created_at, updated_at`,
		c.SellerID, c.Title, c.Description, c.ContractAddress, c.ChainID,
		c.TokenID, c.MetadataURI, c.ImageURL, c.PriceCents, c.Currency,
	).Scan(&c.ID, &c.Status, &c.ViewCount, &c.CreatedAt, &c.UpdatedAt)
}

func (s *CollectibleStore) Update(ctx context.Context, c *model.Collectible) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE collectibles SET title = $2, description = $3, price_cents = $4,
		        image_url = $5, metadata_uri = $6, updated_at = NOW()
		 WHERE id = $1`,
		c.ID, c.Title, c.Description, c.PriceCents, c.ImageURL, c.MetadataURI)
	return err
}

func (s *CollectibleStore) Hide(ctx context.Context, id, hiddenBy uuid.UUID, reason string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE collectibles SET status = 'hidden', hidden_by = $2, hidden_reason = $3, updated_at = NOW()
		 WHERE id = $1`, id, hiddenBy, reason)
	return err
}

func (s *CollectibleStore) Publish(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE collectibles SET status = 'published', hidden_by = NULL, hidden_reason = NULL, updated_at = NOW()
		 WHERE id = $1`, id)
	return err
}

func (s *CollectibleStore) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE collectibles SET view_count = view_count + 1 WHERE id = $1`, id)
	return err
}

func (s *CollectibleStore) CountBySeller(ctx context.Context, sellerID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM collectibles WHERE seller_id = $1 AND status = 'published'`, sellerID).Scan(&count)
	return count, err
}

// Transaction history

func (s *CollectibleStore) GetTxHistory(ctx context.Context, collectibleID uuid.UUID) ([]model.CollectibleTxHistory, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, collectible_id, tx_hash, from_address, to_address, block_number, timestamp
		 FROM collectible_tx_history WHERE collectible_id = $1 ORDER BY timestamp DESC`, collectibleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []model.CollectibleTxHistory
	for rows.Next() {
		var h model.CollectibleTxHistory
		if err := rows.Scan(&h.ID, &h.CollectibleID, &h.TxHash, &h.FromAddress, &h.ToAddress, &h.BlockNumber, &h.Timestamp); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

func (s *CollectibleStore) RecordTxHistory(ctx context.Context, h *model.CollectibleTxHistory) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO collectible_tx_history (collectible_id, tx_hash, from_address, to_address, block_number, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		h.CollectibleID, h.TxHash, h.FromAddress, h.ToAddress, h.BlockNumber, h.Timestamp,
	).Scan(&h.ID)
}
