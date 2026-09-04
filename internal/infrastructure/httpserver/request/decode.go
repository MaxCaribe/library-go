package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/MaxCaribe/library-go/internal/infrastructure/httpserver/response"
)

// DecodeJSON reads a strict JSON body into dst. On failure it writes the error
// response itself and returns false, so a handler can simply return.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(dst)
	if err == nil {
		return true
	}

	var tooLarge *http.MaxBytesError
	var syntax *json.SyntaxError
	var unmarshalType *json.UnmarshalTypeError

	switch {
	case errors.As(err, &tooLarge):
		response.Error(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("request body must not exceed %d bytes", tooLarge.Limit))
	case errors.Is(err, io.EOF):
		response.Error(w, http.StatusBadRequest, "request body is empty")
	case errors.Is(err, io.ErrUnexpectedEOF):
		response.Error(w, http.StatusBadRequest, "request body contains incomplete json")
	case errors.As(err, &syntax):
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("malformed json at character %d", syntax.Offset))
	case errors.As(err, &unmarshalType):
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("field %q must be of type %s", unmarshalType.Field, unmarshalType.Type))
	default:
		response.Error(w, http.StatusBadRequest, "invalid json body")
	}
	return false
}
