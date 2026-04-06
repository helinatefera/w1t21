package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/service"
)

type ABTestHandler struct {
	abtestService *service.ABTestService
}

func NewABTestHandler(as *service.ABTestService) *ABTestHandler {
	return &ABTestHandler{abtestService: as}
}

func (h *ABTestHandler) Create(c echo.Context) error {
	var req dto.CreateABTestRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	createdBy, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	test, err := h.abtestService.Create(c.Request().Context(), req, createdBy)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusCreated, test)
}

func (h *ABTestHandler) List(c echo.Context) error {
	tests, err := h.abtestService.List(c.Request().Context())
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, tests)
}

func (h *ABTestHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid A/B test ID")
	}

	test, err := h.abtestService.GetByID(c.Request().Context(), id)
	if err != nil {
		return mapError(c, err)
	}

	results, _ := h.abtestService.GetResults(c.Request().Context(), id)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"test":    test,
		"results": results,
	})
}

func (h *ABTestHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid A/B test ID")
	}

	var req dto.UpdateABTestRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	test, err := h.abtestService.Update(c.Request().Context(), id, req)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, test)
}

func (h *ABTestHandler) Complete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid A/B test ID")
	}

	if err := h.abtestService.Complete(c.Request().Context(), id); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ABTestHandler) Rollback(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid A/B test ID")
	}

	if err := h.abtestService.Rollback(c.Request().Context(), id); err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ABTestHandler) GetAssignments(c echo.Context) error {
	userID := middleware.GetUserID(c)
	assignments, err := h.abtestService.GetAssignments(c.Request().Context(), userID)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusOK, assignments)
}

func (h *ABTestHandler) GetRegistry(c echo.Context) error {
	return c.JSON(http.StatusOK, service.GetRegistryDTO())
}
