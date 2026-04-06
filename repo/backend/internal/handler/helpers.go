package handler

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
)

var validate = validator.New()

func bindAndValidate(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid request body")
	}
	if err := validate.Struct(req); err != nil {
		return errorResponse(c, http.StatusUnprocessableEntity, dto.CodeValidation, err.Error())
	}
	return nil
}

func errorResponse(c echo.Context, status int, code, message string) error {
	return c.JSON(status, dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: middleware.GetRequestID(c),
		},
	})
}

func mapError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, dto.ErrNotFound):
		return errorResponse(c, http.StatusNotFound, dto.CodeNotFound, err.Error())
	case errors.Is(err, dto.ErrForbidden):
		return errorResponse(c, http.StatusForbidden, dto.CodeForbidden, err.Error())
	case errors.Is(err, dto.ErrUnauthorized):
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, err.Error())
	case errors.Is(err, dto.ErrConflict):
		return errorResponse(c, http.StatusConflict, dto.CodeConflict, err.Error())
	case errors.Is(err, dto.ErrRateLimited):
		return errorResponse(c, http.StatusTooManyRequests, dto.CodeRateLimited, err.Error())
	case errors.Is(err, dto.ErrValidation):
		return errorResponse(c, http.StatusUnprocessableEntity, dto.CodeValidation, err.Error())
	case errors.Is(err, dto.ErrAccountLocked):
		return errorResponse(c, http.StatusLocked, dto.CodeAccountLocked, "account is locked, try again later")
	case errors.Is(err, dto.ErrInvalidCredentials):
		return errorResponse(c, http.StatusUnauthorized, dto.CodeInvalidCredentials, "invalid username or password")
	case errors.Is(err, dto.ErrAttachmentTooLarge):
		return errorResponse(c, http.StatusRequestEntityTooLarge, dto.CodeAttachmentTooLarge, "attachment exceeds 10MB limit")
	case errors.Is(err, dto.ErrDuplicateOrder):
		return errorResponse(c, http.StatusConflict, dto.CodeDuplicateOrder, err.Error())
	case errors.Is(err, dto.ErrInvalidTransition):
		return errorResponse(c, http.StatusUnprocessableEntity, dto.CodeInvalidTransition, err.Error())
	case errors.Is(err, dto.ErrOversold):
		return errorResponse(c, http.StatusConflict, dto.CodeOversold, err.Error())
	default:
		return errorResponse(c, http.StatusInternalServerError, dto.CodeInternal, "internal server error")
	}
}

func pagination(c echo.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return
}

func paginatedResponse(c echo.Context, data interface{}, page, pageSize, total int) error {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return c.JSON(http.StatusOK, dto.PaginatedResponse{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
		TotalPages: totalPages,
	})
}
