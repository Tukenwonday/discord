// Package httpx implements the shared JSON error envelope, response helpers
// and strict request decoding described in section 3 of the contract.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Error codes from the contract.
const (
	CodeValidation      = "validation_error"
	CodeUnauthorized    = "unauthorized"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeInternal        = "internal_error"
	CodePayloadTooLarge = "payload_too_large"
	CodeUnsupportedMedia = "unsupported_media_type"
	CodeRateLimited     = "rate_limited"
)

// DefaultMaxBodyBytes bounds every JSON request body.
const DefaultMaxBodyBytes int64 = 1 << 20

// APIError is the transport level error rendered as the contract envelope.
type APIError struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Fields     map[string]string `json:"fields,omitempty"`
	HTTPStatus int               `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// WithField returns a copy carrying one field level validation message.
func (e *APIError) WithField(field, msg string) *APIError {
	clone := *e
	clone.Fields = map[string]string{}
	for k, v := range e.Fields {
		clone.Fields[k] = v
	}
	clone.Fields[field] = msg
	return &clone
}

// WithStatus overrides the HTTP status of the error.
func (e *APIError) WithStatus(status int) *APIError {
	clone := *e
	clone.HTTPStatus = status
	return &clone
}

// NewValidation builds a validation error with a message and no fields.
func NewValidation(message string) *APIError {
	return &APIError{Code: CodeValidation, Message: message, HTTPStatus: http.StatusBadRequest}
}

// WithFields attaches a whole field map, returning a new error value.
func (e *APIError) WithFields(fields map[string]string) *APIError {
	clone := *e
	clone.Fields = make(map[string]string, len(fields))
	for k, v := range e.Fields {
		clone.Fields[k] = v
	}
	for k, v := range fields {
		clone.Fields[k] = v
	}
	return &clone
}

// NewField builds a validation error carrying a single field message.
func NewField(field, message string) *APIError {
	return NewValidation(message).WithField(field, message)
}

// NewUnauthorized builds a 401 error.
func NewUnauthorized(message string) *APIError {
	return &APIError{Code: CodeUnauthorized, Message: message, HTTPStatus: http.StatusUnauthorized}
}

// NewForbidden builds a 403 error.
func NewForbidden(message string) *APIError {
	return &APIError{Code: CodeForbidden, Message: message, HTTPStatus: http.StatusForbidden}
}

// NewNotFound builds a 404 error.
func NewNotFound(message string) *APIError {
	return &APIError{Code: CodeNotFound, Message: message, HTTPStatus: http.StatusNotFound}
}

// NewConflict builds a 409 error.
func NewConflict(message string) *APIError {
	return &APIError{Code: CodeConflict, Message: message, HTTPStatus: http.StatusConflict}
}

// NewInternal builds a 500 error.
func NewInternal(message string) *APIError {
	return &APIError{Code: CodeInternal, Message: message, HTTPStatus: http.StatusInternalServerError}
}

// NewPayloadTooLarge builds a 413 error.
func NewPayloadTooLarge(message string) *APIError {
	return &APIError{Code: CodePayloadTooLarge, Message: message, HTTPStatus: http.StatusRequestEntityTooLarge}
}

// NewUnsupportedMediaType builds a 415 error.
func NewUnsupportedMediaType(message string) *APIError {
	return &APIError{Code: CodeUnsupportedMedia, Message: message, HTTPStatus: http.StatusUnsupportedMediaType}
}

// NewRateLimited builds a 429 error.
func NewRateLimited(message string) *APIError {
	return &APIError{Code: CodeRateLimited, Message: message, HTTPStatus: http.StatusTooManyRequests}
}

// Status exposes the HTTP status an error renders with.
func (e *APIError) Status() int { return e.HTTPStatus }

// NewInternalFromError wraps a non API error so handlers never leak details.
func NewInternalFromError(err error) *APIError {
	return &APIError{Code: CodeInternal, Message: "internal server error", HTTPStatus: http.StatusInternalServerError}
}

// errorEnvelope is the exact wire shape required by the contract.
type errorEnvelope struct {
	Error *APIError `json:"error"`
}

// RespondJSON writes v with the given status and the JSON content type.
func RespondJSON(c *gin.Context, status int, v any) {
	c.JSON(status, v)
}

// RespondError renders err using the contract envelope, deriving the status
// from the error type when the error is not an APIError.
func RespondError(c *gin.Context, err error) {
	apiErr, ok := err.(*APIError)
	if !ok {
		apiErr = NewInternalFromError(err)
	}
	status := apiErr.HTTPStatus
	if status == 0 {
		status = http.StatusInternalServerError
	}
	c.AbortWithStatusJSON(status, errorEnvelope{Error: apiErr})
}

// RespondValidation renders a 400 validation error with field messages.
func RespondValidation(c *gin.Context, fields map[string]string) {
	apiErr := NewValidation("validation failed")
	if apiErr.Fields == nil {
		apiErr.Fields = fields
	} else {
		for k, v := range fields {
			apiErr.Fields[k] = v
		}
	}
	RespondError(c, apiErr)
}

// DecodeJSON strictly decodes the request body into dst. Unknown fields and
// trailing content are rejected, and oversized bodies yield 413.
func DecodeJSON(c *gin.Context, dst any) error {
	limited := http.MaxBytesReader(c.Writer, c.Request.Body, DefaultMaxBodyBytes)
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return decodeError(c, err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return NewValidation("body must contain a single JSON object")
	}
	return nil
}

func decodeError(c *gin.Context, err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return NewPayloadTooLarge("request body is too large")
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "body"
		}
		return NewField(field, "invalid type")
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return NewField("body", "malformed JSON")
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown field") {
		return NewValidation("unknown field in request body")
	}
	return NewField("body", "invalid JSON")
}