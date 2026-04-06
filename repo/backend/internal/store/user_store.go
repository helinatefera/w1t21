package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledgermint/platform/internal/model"
)

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, display_name, email_encrypted, email_hash,
		        is_locked, locked_until, failed_login_count, created_by, created_at, updated_at
		 FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.EmailEncrypted, &u.EmailHash,
		&u.IsLocked, &u.LockedUntil, &u.FailedLoginCount, &u.CreatedBy, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (s *UserStore) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, display_name, email_encrypted, email_hash,
		        is_locked, locked_until, failed_login_count, created_by, created_at, updated_at
		 FROM users WHERE username = $1`, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.EmailEncrypted, &u.EmailHash,
		&u.IsLocked, &u.LockedUntil, &u.FailedLoginCount, &u.CreatedBy, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (s *UserStore) Create(ctx context.Context, u *model.User) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, display_name, email_encrypted, email_hash, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		u.Username, u.PasswordHash, u.DisplayName, u.EmailEncrypted, u.EmailHash, u.CreatedBy,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (s *UserStore) Update(ctx context.Context, u *model.User) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET display_name = $2, password_hash = $3, email_encrypted = $4,
		        email_hash = $5, updated_at = NOW()
		 WHERE id = $1`,
		u.ID, u.DisplayName, u.PasswordHash, u.EmailEncrypted, u.EmailHash,
	)
	return err
}

func (s *UserStore) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := s.pool.Query(ctx,
		`SELECT id, username, display_name, is_locked, locked_until, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.IsLocked, &u.LockedUntil, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (s *UserStore) IncrementFailedLogin(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET failed_login_count = failed_login_count + 1, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *UserStore) LockAccount(ctx context.Context, id uuid.UUID, until string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET is_locked = TRUE, locked_until = $2, updated_at = NOW() WHERE id = $1`,
		id, until)
	return err
}

func (s *UserStore) UnlockAccount(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET is_locked = FALSE, locked_until = NULL, failed_login_count = 0, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *UserStore) ResetFailedLogin(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE users SET failed_login_count = 0, updated_at = NOW() WHERE id = $1`, id)
	return err
}

// Login attempts (rolling-window tracking)

func (s *UserStore) RecordFailedAttempt(ctx context.Context, userID *uuid.UUID, ip string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO login_attempts (user_id, ip_address) VALUES ($1, $2)`,
		userID, ip)
	return err
}

func (s *UserStore) CountRecentFailuresByUser(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM login_attempts WHERE user_id = $1 AND attempted_at > $2`,
		userID, since).Scan(&count)
	return count, err
}

func (s *UserStore) CountRecentFailuresByIP(ctx context.Context, ip string, since time.Time) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM login_attempts WHERE ip_address = $1 AND attempted_at > $2`,
		ip, since).Scan(&count)
	return count, err
}

func (s *UserStore) ClearFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM login_attempts WHERE user_id = $1`, userID)
	return err
}

func (s *UserStore) CleanupOldLoginAttempts(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM login_attempts WHERE attempted_at < NOW() - INTERVAL '24 hours'`)
	return err
}

// Roles

func (s *UserStore) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT ur.user_id, ur.role_id, r.name, ur.granted_by, ur.granted_at
		 FROM user_roles ur JOIN roles r ON ur.role_id = r.id
		 WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []model.UserRole
	for rows.Next() {
		var r model.UserRole
		if err := rows.Scan(&r.UserID, &r.RoleID, &r.RoleName, &r.GrantedBy, &r.GrantedAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, nil
}

func (s *UserStore) AddRole(ctx context.Context, userID uuid.UUID, roleName string, grantedBy uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, granted_by)
		 SELECT $1, id, $3 FROM roles WHERE name = $2
		 ON CONFLICT DO NOTHING`, userID, roleName, grantedBy)
	return err
}

func (s *UserStore) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID)
	return err
}

func (s *UserStore) GetRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var r model.Role
	err := s.pool.QueryRow(ctx, `SELECT id, name, description FROM roles WHERE name = $1`, name).
		Scan(&r.ID, &r.Name, &r.Description)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// AdminExists reports whether at least one user with the administrator role exists.
func (s *UserStore) AdminExists(ctx context.Context) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM user_roles ur
			JOIN roles r ON ur.role_id = r.id
			WHERE r.name = 'administrator'
		)`).Scan(&exists)
	return exists, err
}

// Refresh tokens

func (s *UserStore) CreateRefreshToken(ctx context.Context, rt *model.RefreshToken) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, family_id, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		rt.ID, rt.UserID, rt.TokenHash, rt.FamilyID, rt.ExpiresAt)
	return err
}

func (s *UserStore) GetRefreshTokenByHash(ctx context.Context, hash []byte) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, hash).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.FamilyID, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &rt, err
}

func (s *UserStore) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *UserStore) RevokeRefreshTokenFamily(ctx context.Context, familyID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE family_id = $1 AND revoked_at IS NULL`, familyID)
	return err
}

func (s *UserStore) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func (s *UserStore) CleanupExpiredTokens(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '30 days'`)
	return err
}

// IP Rules

func (s *UserStore) GetAllIPRules() ([]model.IPRule, error) {
	rows, err := s.pool.Query(context.Background(),
		`SELECT id, cidr::text, action, created_by, created_at FROM ip_rules ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.IPRule
	for rows.Next() {
		var r model.IPRule
		if err := rows.Scan(&r.ID, &r.CIDR, &r.Action, &r.CreatedBy, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *UserStore) CreateIPRule(ctx context.Context, r *model.IPRule) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO ip_rules (cidr, action, created_by) VALUES ($1::cidr, $2, $3) RETURNING id, created_at`,
		r.CIDR, r.Action, r.CreatedBy).Scan(&r.ID, &r.CreatedAt)
}

func (s *UserStore) DeleteIPRule(ctx context.Context, id uuid.UUID) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM ip_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("ip rule not found")
	}
	return nil
}
