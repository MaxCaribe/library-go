package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MaxCaribe/library-go/internal/domain"
)

// dataEnvelope wraps a single value in a {"data": ...} JSON object.
type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

func WithData[T any](w http.ResponseWriter, status int, data T) {
	JSON(w, status, dataEnvelope[T]{Data: data})
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

func ValidationError(w http.ResponseWriter, fields map[string]string) {
	JSON(w, http.StatusBadRequest, map[string]any{
		"error":  "validation failed",
		"fields": fields,
	})
}

// DomainError maps domain sentinel errors to HTTP responses.
// Returns true if it wrote a response (caller should return), false if the error is unrecognized.
// Every new sentinel in internal/domain belongs in this switch, or handlers will report it as a 500.
func DomainError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		Error(w, http.StatusNotFound, "Not found")
		return true
	case errors.Is(err, domain.ErrAccessDenied):
		Error(w, http.StatusForbidden, "Access denied")
		return true
	case errors.Is(err, domain.ErrAlreadyExists):
		Error(w, http.StatusConflict, err.Error())
		return true
	case errors.Is(err, domain.ErrInvalidInput):
		Error(w, http.StatusBadRequest, err.Error())
		return true
	}
	return false
}
