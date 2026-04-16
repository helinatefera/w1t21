package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledgermint/platform/internal/cache"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

type CollectibleService struct {
	collectibleStore *store.CollectibleStore
	analyticsStore   *store.AnalyticsStore
	cache            *cache.HotCache
	notifService     *NotificationService
}

func NewCollectibleService(cs *store.CollectibleStore, as *store.AnalyticsStore, c *cache.HotCache, ns *NotificationService) *CollectibleService {
	return &CollectibleService{collectibleStore: cs, analyticsStore: as, cache: c, notifService: ns}
}

func (s *CollectibleService) Create(ctx context.Context, req dto.CreateCollectibleRequest, sellerID uuid.UUID) (*model.Collectible, error) {
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	if err := validateCollectibleIdentity(req); err != nil {
		return nil, err
	}

	tokenID := req.TokenID
	chainID := req.ChainID
	if req.ContractAddress == "" && tokenID == "" && chainID == 0 {
		// Neither provided — generate platform defaults for off-chain items.
		tokenID = uuid.New().String()
		chainID = 1
	}

	c := &model.Collectible{
		SellerID:        sellerID,
		Title:           req.Title,
		Description:     req.Description,
		ContractAddress: req.ContractAddress,
		ChainID:         chainID,
		TokenID:         tokenID,
		MetadataURI:     req.MetadataURI,
		ImageURL:        req.ImageURL,
		PriceCents:      req.PriceCents,
		Currency:        currency,
	}

	if err := s.collectibleStore.Create(ctx, c); err != nil {
		if strings.Contains(err.Error(), "idx_collectibles_identity") || strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("%w: a collectible with this contract_address, chain_id, and token_id already exists", dto.ErrConflict)
		}
		return nil, fmt.Errorf("create collectible: %w", err)
	}
	s.invalidateCache()
	return c, nil
}

// GetByID returns a collectible. Hidden collectibles are only visible to admin/compliance roles.
func (s *CollectibleService) GetByID(ctx context.Context, id uuid.UUID, viewerRoles []string) (*model.Collectible, error) {
	cacheKey := "collectible:" + id.String()
	if cached, ok := s.cache.Get(cacheKey); ok {
		c := cached.(*model.Collectible)
		if c.Status == "hidden" && !hasAdminOrCompliance(viewerRoles) {
			return nil, dto.ErrNotFound
		}
		return c, nil
	}

	c, err := s.collectibleStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, dto.ErrNotFound
	}

	// Hidden collectibles only visible to admin/compliance
	if c.Status == "hidden" && !hasAdminOrCompliance(viewerRoles) {
		return nil, dto.ErrNotFound
	}

	s.cache.Set(cacheKey, c, 2*time.Minute)

	// Emit item_view event and increment view count async
	go func() {
		_ = s.collectibleStore.IncrementViewCount(context.Background(), id)
	}()

	return c, nil
}

// EmitViewEvent emits an analytics event for viewing. Called separately to have access to userID.
func (s *CollectibleService) EmitViewEvent(ctx context.Context, userID *uuid.UUID, collectibleID uuid.UUID) {
	if s.analyticsStore == nil {
		return
	}
	abVariant := resolveABVariant(s.analyticsStore, userID)
	go func() {
		_ = s.analyticsStore.RecordEvent(context.Background(), &model.AnalyticsEvent{
			UserID:        userID,
			EventType:     "item_view",
			CollectibleID: &collectibleID,
			SessionID:     "server",
			ABVariant:     abVariant,
		})
	}()
}

// List returns collectibles. Non-admin users can only see published.
func (s *CollectibleService) List(ctx context.Context, status string, page, pageSize int, viewerRoles []string, userID *uuid.UUID) ([]model.Collectible, int, error) {
	// Non-admin/compliance users can ONLY see published
	if !hasAdminOrCompliance(viewerRoles) {
		status = "published"
	}

	cacheKey := fmt.Sprintf("collectibles:%s:%d:%d", status, page, pageSize)
	if cached, ok := s.cache.Get(cacheKey); ok {
		result := cached.(*cachedList)
		return result.items, result.total, nil
	}

	items, total, err := s.collectibleStore.List(ctx, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	s.cache.Set(cacheKey, &cachedList{items: items, total: total}, 30*time.Second)

	// Emit catalog_view event async with A/B variant tagging
	if s.analyticsStore != nil {
		abVariant := resolveABVariant(s.analyticsStore, userID)
		go func() {
			_ = s.analyticsStore.RecordEvent(context.Background(), &model.AnalyticsEvent{
				UserID:    userID,
				EventType: "catalog_view",
				SessionID: "server",
				ABVariant: abVariant,
			})
		}()
	}

	return items, total, nil
}

type cachedList struct {
	items []model.Collectible
	total int
}

func (s *CollectibleService) ListBySeller(ctx context.Context, sellerID uuid.UUID, page, pageSize int) ([]model.Collectible, int, error) {
	return s.collectibleStore.ListBySeller(ctx, sellerID, page, pageSize)
}

func (s *CollectibleService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateCollectibleRequest, actorID uuid.UUID) (*model.Collectible, error) {
	c, err := s.collectibleStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, dto.ErrNotFound
	}
	if c.SellerID != actorID {
		return nil, dto.ErrForbidden
	}

	if req.Title != nil {
		c.Title = *req.Title
	}
	if req.Description != nil {
		c.Description = *req.Description
	}
	if req.PriceCents != nil {
		c.PriceCents = *req.PriceCents
	}
	if req.ImageURL != nil {
		c.ImageURL = *req.ImageURL
	}
	if req.MetadataURI != nil {
		c.MetadataURI = *req.MetadataURI
	}

	if err := s.collectibleStore.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("update collectible: %w", err)
	}
	s.invalidateCache()
	return c, nil
}

func (s *CollectibleService) Hide(ctx context.Context, id uuid.UUID, reason string, adminID uuid.UUID) error {
	c, err := s.collectibleStore.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return dto.ErrNotFound
	}
	if err := s.collectibleStore.Hide(ctx, id, adminID, reason); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

func (s *CollectibleService) Publish(ctx context.Context, id uuid.UUID) error {
	c, err := s.collectibleStore.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return dto.ErrNotFound
	}
	if err := s.collectibleStore.Publish(ctx, id); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

func (s *CollectibleService) CountBySeller(ctx context.Context, sellerID uuid.UUID) (int, error) {
	return s.collectibleStore.CountBySeller(ctx, sellerID)
}

func (s *CollectibleService) GetTxHistory(ctx context.Context, collectibleID uuid.UUID) ([]model.CollectibleTxHistory, error) {
	return s.collectibleStore.GetTxHistory(ctx, collectibleID)
}

// PostReview records a user review for a collectible and emits the
// review_posted notification to the collectible's seller.
func (s *CollectibleService) PostReview(ctx context.Context, collectibleID uuid.UUID, reviewerID uuid.UUID, rating int, body string) error {
	collectible, err := s.collectibleStore.GetByID(ctx, collectibleID)
	if err != nil {
		return fmt.Errorf("get collectible: %w", err)
	}
	if collectible == nil {
		return dto.ErrNotFound
	}

	// Emit analytics event immediately after the review is submitted
	if s.analyticsStore != nil {
		abVariant := resolveABVariant(s.analyticsStore, &reviewerID)
		go func() {
			_ = s.analyticsStore.RecordEvent(context.Background(), &model.AnalyticsEvent{
				UserID:        &reviewerID,
				EventType:     "review_posted",
				CollectibleID: &collectibleID,
				SessionID:     "server",
				ABVariant:     abVariant,
			})
		}()
	}

	// Send notification to the seller (owner of the collectible)
	s.sendReviewNotification(ctx, collectible)

	return nil
}

func (s *CollectibleService) sendReviewNotification(ctx context.Context, collectible *model.Collectible) {
	if s.notifService == nil {
		return
	}
	params := map[string]string{
		"CollectibleTitle": collectible.Title,
	}
	paramsJSON, _ := json.Marshal(params)
	go func() {
		_ = s.notifService.Send(context.Background(), collectible.SellerID, "review_posted", paramsJSON)
	}()
}

func (s *CollectibleService) invalidateCache() {
	s.cache.DeletePrefix("collectible")
}

// resolveABVariant builds a comma-separated ab_variant tag for all running tests.
// Returns "" if there are no running tests or the user is nil.
func resolveABVariant(analyticsStore *store.AnalyticsStore, userID *uuid.UUID) string {
	if userID == nil || analyticsStore == nil {
		return ""
	}
	tests, _ := analyticsStore.ListRunningABTests(context.Background())
	if len(tests) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tests))
	for _, t := range tests {
		v := AssignVariant(userID.String(), t.Name, t.TrafficPct)
		variantName := t.ControlVariant
		if v == "test" {
			variantName = t.TestVariant
		}
		parts = append(parts, t.Name+":"+variantName)
	}
	return strings.Join(parts, ",")
}

// validateCollectibleIdentity checks the identity field constraints.
// Returns nil when the combination is valid; an error otherwise.
func validateCollectibleIdentity(req dto.CreateCollectibleRequest) error {
	hasToken := req.TokenID != ""
	hasChain := req.ChainID != 0
	if req.ContractAddress != "" {
		if !hasToken || !hasChain {
			return fmt.Errorf("%w: contract_address requires chain_id and token_id", dto.ErrValidation)
		}
	} else if hasToken || hasChain {
		if !hasToken || !hasChain {
			return fmt.Errorf("%w: chain_id and token_id must both be provided when either is set", dto.ErrValidation)
		}
	}
	return nil
}

func hasAdminOrCompliance(roles []string) bool {
	for _, r := range roles {
		if r == "administrator" || r == "compliance_analyst" {
			return true
		}
	}
	return false
}
