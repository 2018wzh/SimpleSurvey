package apperror

import "net/http"

type AppError struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Status  int               `json:"-"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(status, code int, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, 400, message)
}

func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, 401, message)
}

func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, 403, message)
}

func NotFound(message string) *AppError {
	return New(http.StatusNotFound, 404, message)
}

func Conflict(message string) *AppError {
	return New(http.StatusConflict, 409, message)
}

func PreconditionFailed(message string) *AppError {
	return New(http.StatusPreconditionFailed, 412, message)
}

func Internal(message string) *AppError {
	return New(http.StatusInternalServerError, 500, message)
}

func WithDetails(err *AppError, details map[string]string) *AppError {
	err.Details = details
	return err
}
