package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ledgermint/platform/internal/crypto"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

type UserService struct {
	userStore *store.UserStore
	encryptor *crypto.AESEncryptor
}

func NewUserService(userStore *store.UserStore, encryptor *crypto.AESEncryptor) *UserService {
	return &UserService{userStore: userStore, encryptor: encryptor}
}

func (s *UserService) Create(ctx context.Context, req dto.CreateUserRequest, createdBy uuid.UUID) (*model.User, error) {
	existing, err := s.userStore.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if existing != nil {
		return nil, dto.ErrConflict
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		DisplayName:  req.DisplayName,
		CreatedBy:    &createdBy,
	}

	if req.Email != "" {
		encrypted, err := s.encryptor.Encrypt([]byte(req.Email))
		if err != nil {
			return nil, fmt.Errorf("encrypt email: %w", err)
		}
		user.EmailEncrypted = encrypted
		user.EmailHash = s.encryptor.HMACHash([]byte(req.Email))
	}

	if err := s.userStore.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Auto-assign buyer and seller roles to every new user
	_ = s.userStore.AddRole(ctx, user.ID, "buyer", createdBy)
	_ = s.userStore.AddRole(ctx, user.ID, "seller", createdBy)

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.userStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, dto.ErrNotFound
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	return s.userStore.List(ctx, page, pageSize)
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) (*model.User, error) {
	user, err := s.userStore.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, dto.ErrNotFound
	}

	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.Password != nil {
		hash, err := crypto.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		user.PasswordHash = hash
	}
	if req.Email != nil {
		encrypted, err := s.encryptor.Encrypt([]byte(*req.Email))
		if err != nil {
			return nil, fmt.Errorf("encrypt email: %w", err)
		}
		user.EmailEncrypted = encrypted
		user.EmailHash = s.encryptor.HMACHash([]byte(*req.Email))
	}

	if err := s.userStore.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

func (s *UserService) AddRole(ctx context.Context, userID uuid.UUID, roleName string, grantedBy uuid.UUID) error {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return dto.ErrNotFound
	}
	return s.userStore.AddRole(ctx, userID, roleName, grantedBy)
}

func (s *UserService) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.userStore.RemoveRole(ctx, userID, roleID)
}

func (s *UserService) GetRoles(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error) {
	return s.userStore.GetUserRoles(ctx, userID)
}

func (s *UserService) UnlockAccount(ctx context.Context, userID uuid.UUID) error {
	if err := s.userStore.UnlockAccount(ctx, userID); err != nil {
		return err
	}
	return s.userStore.ClearFailedAttempts(ctx, userID)
}

func (s *UserService) GetMaskedEmail(ctx context.Context, user *model.User) string {
	if user.EmailEncrypted == nil {
		return ""
	}
	plaintext, err := s.encryptor.Decrypt(user.EmailEncrypted)
	if err != nil {
		return "***"
	}
	email := string(plaintext)
	return maskEmail(email)
}

func maskEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "***"
	}
	masked := string(email[0])
	for i := 1; i < at; i++ {
		masked += "*"
	}
	masked += email[at:]
	return masked
}
