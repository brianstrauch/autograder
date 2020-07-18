package main

import (
	"net/http"
)

const problemsDirErr = "PROBLEMS_DIR must be set"

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
