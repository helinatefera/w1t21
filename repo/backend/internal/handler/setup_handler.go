package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/crypto"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/model"
	"github.com/ledgermint/platform/internal/store"
)

type SetupHandler struct {
	userStore *store.UserStore
}

func NewSetupHandler(us *store.UserStore) *SetupHandler {
	return &SetupHandler{userStore: us}
}

// Status returns whether initial setup has been completed (an admin user exists).
func (h *SetupHandler) Status(c echo.Context) error {
	exists, err := h.userStore.AdminExists(c.Request().Context())
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, dto.CodeInternal, "internal server error")
	}
	return c.JSON(http.StatusOK, map[string]bool{"setup_complete": exists})
}

// Bootstrap creates the first administrator account. It only works when no
// admin user exists — once an admin has been created this endpoint returns 409.
func (h *SetupHandler) Bootstrap(c echo.Context) error {
	exists, err := h.userStore.AdminExists(c.Request().Context())
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, dto.CodeInternal, "internal server error")
	}
	if exists {
		return errorResponse(c, http.StatusConflict, dto.CodeConflict, "setup already complete: an administrator account exists")
	}

	var req dto.BootstrapAdminRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, dto.CodeInternal, "internal server error")
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: hash,
		DisplayName:  req.DisplayName,
	}

	if err := h.userStore.Create(c.Request().Context(), user); err != nil {
		return errorResponse(c, http.StatusInternalServerError, dto.CodeInternal, "failed to create admin user")
	}

	// The first admin grants itself the administrator role.
	if err := h.userStore.AddRole(c.Request().Context(), user.ID, "administrator", user.ID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, dto.CodeInternal, "failed to assign administrator role")
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"status":  "ok",
		"user_id": user.ID.String(),
		"message": "Administrator account created. You may now log in.",
	})
}
