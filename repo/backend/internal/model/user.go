package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID  `json:"id"`
	Username         string     `json:"username"`
	PasswordHash     string     `json:"-"`
	DisplayName      string     `json:"display_name"`
	EmailEncrypted   []byte     `json:"-"`
	EmailHash        []byte     `json:"-"`
	IsLocked         bool       `json:"is_locked"`
	LockedUntil      *time.Time `json:"locked_until,omitempty"`
	FailedLoginCount int        `json:"-"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type UserRole struct {
	UserID    uuid.UUID `json:"user_id"`
	RoleID    uuid.UUID `json:"role_id"`
	RoleName  string    `json:"role_name"`
	GrantedBy uuid.UUID `json:"granted_by"`
	GrantedAt time.Time `json:"granted_at"`
}

type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash []byte     `json:"-"`
	FamilyID  uuid.UUID  `json:"family_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
