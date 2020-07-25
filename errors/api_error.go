package errors

import (
	"encoding/json"
	"log"
	"net/http"
)

const ProblemsDirErr = "PROBLEMS_DIR must be set"

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"error,omitempty"`
}

func NewInternalError(err error) *APIError {
	return &APIError{
		Code:    http.StatusInternalServerError,
		Message: "Internal Error",
		Err:     err,
	}
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request) *APIError

func (h ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path)

	if err := h(w, r); err != nil {
		log.Println(err.Code, err.Message)
		if err.Err != nil {
			log.Println(err.Err)
		}

		w.WriteHeader(err.Code)
		err.Err = nil
		if err := json.NewEncoder(w).Encode(err); err != nil {
			log.Println(err)
		}
	}
}
