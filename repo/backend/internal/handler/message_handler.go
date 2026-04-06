package handler

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ledgermint/platform/internal/dto"
	"github.com/ledgermint/platform/internal/middleware"
	"github.com/ledgermint/platform/internal/service"
)

type MessageHandler struct {
	messageService *service.MessageService
}

func NewMessageHandler(ms *service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: ms}
}

func (h *MessageHandler) List(c echo.Context) error {
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid order ID")
	}

	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	page, pageSize := pagination(c)
	messages, total, err := h.messageService.ListByOrder(c.Request().Context(), orderID, userID, page, pageSize)
	if err != nil {
		return mapError(c, err)
	}
	return paginatedResponse(c, messages, page, pageSize, total)
}

func (h *MessageHandler) Send(c echo.Context) error {
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid order ID")
	}

	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	body := c.FormValue("body")
	if body == "" {
		return errorResponse(c, http.StatusUnprocessableEntity, dto.CodeValidation, "message body is required")
	}
	if len(body) > 10000 {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "message body exceeds 10000 character limit")
	}

	// Server-side PII detection
	if detected, piiTypes := service.DetectPII(body); detected {
		return errorResponse(c, http.StatusUnprocessableEntity, dto.CodeValidation, service.PIIErrorMessage(piiTypes))
	}

	var attachmentData []byte
	var attachmentMime string
	var attachmentSize int

	file, err := c.FormFile("attachment")
	if err == nil {
		// Server-side 10MB enforcement
		if file.Size > service.MaxAttachmentSize {
			return errorResponse(c, http.StatusRequestEntityTooLarge, dto.CodeAttachmentTooLarge, "attachment exceeds 10MB limit")
		}

		src, err := file.Open()
		if err != nil {
			return errorResponse(c, http.StatusInternalServerError, dto.CodeInternal, "failed to read attachment")
		}
		defer src.Close()

		mime := file.Header.Get("Content-Type")
		ext := filepath.Ext(file.Filename)

		// Read the full attachment into memory before persisting so we can
		// inspect it for PII regardless of format.
		buf := &bytes.Buffer{}
		if _, err := io.Copy(buf, src); err != nil {
			return errorResponse(c, http.StatusInternalServerError, dto.CodeInternal, "failed to read attachment")
		}

		// Reject file types that are neither scannable text nor an
		// approved binary type (images, PDF, SVG).
		if !isTextAttachment(mime, ext) && !isSafeBinaryAttachment(mime, ext) {
			return errorResponse(c, http.StatusUnprocessableEntity, dto.CodeValidation,
				"attachment type not allowed: "+mime+" ("+ext+"); supported types: images, PDF, plain text, CSV")
		}

		// PII scan — applied to ALL attachments regardless of format.
		scannableText := service.ExtractScannableText(buf.Bytes(), mime, ext)
		if detected, piiTypes := service.DetectPII(scannableText); detected {
			c.Logger().Warn("attachment PII blocked for order ", orderID.String())
			return errorResponse(c, http.StatusUnprocessableEntity, dto.CodeValidation,
				"attachment blocked: contains sensitive personal information ("+strings.Join(piiTypes, ", ")+")")
		}

		attachmentData = buf.Bytes()
		attachmentSize = int(file.Size)
		attachmentMime = mime
	}

	msg, err := h.messageService.Send(c.Request().Context(), orderID, userID, body,
		attachmentData, attachmentSize, attachmentMime)
	if err != nil {
		return mapError(c, err)
	}
	return c.JSON(http.StatusCreated, msg)
}

// DownloadAttachment serves an attachment by message ID directly from PostgreSQL.
// The binary data is stored in the message_attachments table — no filesystem
// paths are involved, making PostgreSQL the sole system of record.
func (h *MessageHandler) DownloadAttachment(c echo.Context) error {
	msgID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, dto.CodeValidation, "invalid message ID")
	}

	userID, err := uuid.Parse(middleware.GetUserID(c))
	if err != nil {
		return errorResponse(c, http.StatusUnauthorized, dto.CodeUnauthorized, "invalid user")
	}

	att, err := h.messageService.GetAttachment(c.Request().Context(), msgID, userID)
	if err != nil {
		return mapError(c, err)
	}

	return c.Blob(http.StatusOK, att.Mime, att.Data)
}

// isTextAttachment returns true for file types whose content should be scanned as text.
func isTextAttachment(mime string, ext string) bool {
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	ext = strings.ToLower(ext)
	return ext == ".csv" || ext == ".txt"
}

// safeBinaryMIMEs lists binary MIME types that are allowed as attachments.
var safeBinaryMIMEs = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/svg+xml":   true,
	"application/pdf": true,
}

// safeBinaryExts provides a fallback when the MIME type from the client
// is generic (e.g. application/octet-stream).
var safeBinaryExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".svg": true, ".pdf": true,
}

// isSafeBinaryAttachment returns true for binary file types that are
// explicitly allowed.
func isSafeBinaryAttachment(mime string, ext string) bool {
	if safeBinaryMIMEs[mime] {
		return true
	}
	return safeBinaryExts[strings.ToLower(ext)]
}
