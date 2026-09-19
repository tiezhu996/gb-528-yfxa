package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *AppError) Unwrap() error { return e.Cause }

func BadRequest(code, message string, details any) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: code, Message: message, Details: details}
}

func Unauthorized(code, message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Code: code, Message: message}
}

func Forbidden(code, message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: code, Message: message}
}

func NotFound(code, message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: code, Message: message}
}

func Conflict(code, message string, cause error) *AppError {
	return &AppError{Status: http.StatusConflict, Code: code, Message: message, Cause: cause}
}

func Unprocessable(code, message string, details any) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: code, Message: message, Details: details}
}

func Internal(cause error) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "the request could not be completed", Cause: cause}
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data, "request_id": RequestID(c)})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data, "request_id": RequestID(c)})
}

func Page(c *gin.Context, data any, page, pageSize int, total int64) {
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": gin.H{"page": page, "page_size": pageSize, "total": total}, "request_id": RequestID(c)})
}

func Fail(c *gin.Context, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = Internal(err)
	}
	payload := gin.H{"code": appErr.Code, "message": appErr.Message}
	if appErr.Details != nil {
		payload["details"] = appErr.Details
	}
	c.AbortWithStatusJSON(appErr.Status, gin.H{"error": payload, "request_id": RequestID(c)})
}

func RequestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}

func ParseID(c *gin.Context, name string) (uint, error) {
	raw := c.Param(name)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return 0, BadRequest("INVALID_ID", name+" must be a positive integer", nil)
	}
	return uint(value), nil
}

func Pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func SummaryJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{\"summary\":\"unavailable\"}"
	}
	return string(encoded)
}
