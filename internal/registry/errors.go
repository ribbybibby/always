package registry

import (
	"encoding/json"
	"log"
	"net/http"
)

var (
	errReadOnly = &Error{
		Status:  http.StatusMethodNotAllowed,
		Code:    "DENIED",
		Message: "read-only",
	}
)

// ErrorResponse is the response body of an error
type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

// Error is an error returned by a registry
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

// Error implements error
func (e *Error) Error() string {
	return e.Message
}

func newError(err error) error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_ERROR",
		Message: err.Error(),
	}
}

func serveError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if e, ok := err.(*Error); ok {
		w.WriteHeader(e.Status)

		resp := &ErrorResponse{
			Errors: []Error{*e},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("ERROR writing body: %+v: %v", e, err)
		}

		return
	}

	http.Error(w, err.Error(), http.StatusInternalServerError)
}
