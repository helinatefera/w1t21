package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/service"
	"github.com/ledgermint/platform/internal/store"
)

type UserHandler struct {
	userService *service.UserService
	auditStore  *store.AuditStore
}

func NewUserHandler(us *service.UserService, audit *store.AuditStore) *UserHandler {
	return &UserHandler{userService: us, auditStore: audit}
}

func (h *UserHandler) Create(c echo.Context) error {
	var req dto.CreateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	createdBy, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	user, err := h.userService.Create(c.Request().Context(), req, createdBy)
	if err != nil {
		return mapError(c, err)
	}

	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &createdBy, model.AuditActionUserCreate, "user", &user.ID,
			map[string]interface{}{"username": user.Username}, c.RealIP())
	}
	return c.JSON(http.StatusCreated, dto.UserResponse{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
		IsLocked: user.IsLocked, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	})
}

func (h *UserHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid user ID")
	}

	user, err := h.userService.GetByID(c.Request().Context(), id)
	if err != nil {
		return mapError(c, err)
	}

	email := h.userService.GetMaskedEmail(c.Request().Context(), user)
	return c.JSON(http.StatusOK, dto.UserResponse{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
		Email: email, IsLocked: user.IsLocked, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	})
}

func (h *UserHandler) List(c echo.Context) error {
	page, pageSize := pagination(c)
	users, total, err := h.userService.List(c.Request().Context(), page, pageSize)
	if err != nil {
		return mapError(c, err)
	}

	resp := make([]dto.UserResponse, len(users))
	for i, u := range users {
		resp[i] = dto.UserResponse{
			ID: u.ID, Username: u.Username, DisplayName: u.DisplayName,
			IsLocked: u.IsLocked, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		}
	}
	return paginatedResponse(c, resp, page, pageSize, total)
}

func (h *UserHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid user ID")
	}

	var req dto.UpdateUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := h.userService.Update(c.Request().Context(), id, req)
	if err != nil {
		return mapError(c, err)
	}

	return c.JSON(http.StatusOK, dto.UserResponse{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
		IsLocked: user.IsLocked, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	})
}

func (h *UserHandler) AddRole(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid user ID")
	}

	var req dto.AddRoleRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	grantedBy, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	if err := h.userService.AddRole(c.Request().Context(), userID, req.RoleName, grantedBy); err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &grantedBy, model.AuditActionRoleAdd, "user", &userID,
			map[string]interface{}{"role": req.RoleName}, c.RealIP())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *UserHandler) RemoveRole(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid user ID")
	}
	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid role ID")
	}

	actorID, _ := uuid.Parse(middleware.GetUserID(c))
	if err := h.userService.RemoveRole(c.Request().Context(), userID, roleID); err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &actorID, model.AuditActionRoleRemove, "user", &userID,
			map[string]interface{}{"role_id": roleID.String()}, c.RealIP())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *UserHandler) Unlock(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid user ID")
	}

	actorID, _ := uuid.Parse(middleware.GetUserID(c))
	if err := h.userService.UnlockAccount(c.Request().Context(), userID); err != nil {
		return mapError(c, err)
	}
	if h.auditStore != nil {
		h.auditStore.LogEvent(c.Request().Context(), &actorID, model.AuditActionUserUnlock, "user", &userID, nil, c.RealIP())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
