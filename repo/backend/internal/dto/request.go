package dto

import "github.com/google/uuid"

type BootstrapAdminRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=100"`
	Password    string `json:"password" validate:"required,min=12,max=128"`
	DisplayName string `json:"display_name" validate:"required,max=200"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=100"`
	Password string `json:"password" validate:"required,min=8"`
}

type CreateUserRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=100"`
	Password    string `json:"password" validate:"required,min=8,max=128"`
	DisplayName string `json:"display_name" validate:"required,max=200"`
	Email       string `json:"email" validate:"omitempty,email"`
}

type UpdateUserRequest struct {
	DisplayName *string `json:"display_name" validate:"omitempty,max=200"`
	Password    *string `json:"password" validate:"omitempty,min=8,max=128"`
	Email       *string `json:"email" validate:"omitempty,email"`
}

type AddRoleRequest struct {
	RoleName string `json:"role_name" validate:"required,oneof=buyer seller administrator compliance_analyst"`
}

type CreateCollectibleRequest struct {
	Title           string `json:"title" validate:"required,max=300"`
	Description     string `json:"description" validate:"max=5000"`
	ContractAddress string `json:"contract_address" validate:"omitempty,max=42"`
	ChainID         int    `json:"chain_id" validate:"omitempty,min=1"`
	TokenID         string `json:"token_id" validate:"omitempty,max=78"`
	MetadataURI     string `json:"metadata_uri" validate:"omitempty,url"`
	ImageURL        string `json:"image_url" validate:"omitempty,url"`
	PriceCents      int64  `json:"price_cents" validate:"required,min=1"`
	Currency        string `json:"currency" validate:"omitempty,len=3"`
}

type UpdateCollectibleRequest struct {
	Title       *string `json:"title" validate:"omitempty,max=300"`
	Description *string `json:"description" validate:"omitempty,max=5000"`
	PriceCents  *int64  `json:"price_cents" validate:"omitempty,min=1"`
	ImageURL    *string `json:"image_url" validate:"omitempty,url"`
	MetadataURI *string `json:"metadata_uri" validate:"omitempty,url"`
}

type HideCollectibleRequest struct {
	Reason string `json:"reason" validate:"required,max=500"`
}

type CreateOrderRequest struct {
	CollectibleID uuid.UUID `json:"collectible_id" validate:"required"`
}

type UpdateFulfillmentRequest struct {
	Carrier        string `json:"carrier" validate:"omitempty,max=100"`
	TrackingNumber string `json:"tracking_number" validate:"omitempty,max=200"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason" validate:"required,max=500"`
}

type SendMessageRequest struct {
	Body string `json:"body" validate:"required,max=10000"`
}

type UpdateNotificationPrefsRequest struct {
	Preferences      map[string]bool `json:"preferences" validate:"required"`
	SubscriptionMode string          `json:"subscription_mode" validate:"omitempty,oneof=status_only all_events"`
}

type CreateABTestRequest struct {
	Name                 string `json:"name" validate:"required,max=200"`
	Description          string `json:"description" validate:"max=1000"`
	TrafficPct           int    `json:"traffic_pct" validate:"required,min=1,max=100"`
	StartDate            string `json:"start_date" validate:"required"`
	EndDate              string `json:"end_date" validate:"required"`
	ControlVariant       string `json:"control_variant" validate:"required,max=100"`
	TestVariant          string `json:"test_variant" validate:"required,max=100"`
	RollbackThresholdPct int    `json:"rollback_threshold_pct" validate:"required,min=1,max=100"`
}

type UpdateABTestRequest struct {
	Description          *string `json:"description" validate:"omitempty,max=1000"`
	TrafficPct           *int    `json:"traffic_pct" validate:"omitempty,min=1,max=100"`
	EndDate              *string `json:"end_date"`
	RollbackThresholdPct *int    `json:"rollback_threshold_pct" validate:"omitempty,min=1,max=100"`
}

type CreateIPRuleRequest struct {
	CIDR   string `json:"cidr" validate:"required,cidrv4|cidrv6"`
	Action string `json:"action" validate:"required,oneof=allow deny"`
}

type ApproveRefundRequest struct {
	Reason string `json:"reason" validate:"required,max=500"`
}

type OpenArbitrationRequest struct {
	Reason string `json:"reason" validate:"required,max=1000"`
}

type PostReviewRequest struct {
	CollectibleID uuid.UUID `json:"collectible_id" validate:"required"`
	Rating        int       `json:"rating" validate:"required,min=1,max=5"`
	Body          string    `json:"body" validate:"required,max=5000"`
}

type RecordEventRequest struct {
	EventType     string     `json:"event_type" validate:"required,max=50"`
	CollectibleID *uuid.UUID `json:"collectible_id"`
	SessionID     string     `json:"session_id" validate:"required,max=64"`
	ABVariant     string     `json:"ab_variant" validate:"omitempty,max=100"`
}
