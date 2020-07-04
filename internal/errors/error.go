package errors

import (
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
