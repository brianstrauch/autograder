package errors

import (
	"encoding/json"
	"log"
	"net/http"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"error,omitempty"`
}

func NewInternalError(err error) *Error {
	return &Error{
		Code:    http.StatusInternalServerError,
		Message: "Internal Error",
		Err:     err,
	}
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request) *Error

func (h ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		w.WriteHeader(err.Code)
		err.Err = nil
		if err := json.NewEncoder(w).Encode(err); err != nil {
			log.Println(err)
		}
	}
}
