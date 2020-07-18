package errors

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomHandler(t *testing.T) {
	unsafeEndpoint := func(w http.ResponseWriter, r *http.Request) *APIError {
		return &APIError{
			Code:    http.StatusInternalServerError,
			Message: "Something bad happened.",
			Err:     errors.New(""),
		}
	}
	handler := ErrorHandler(unsafeEndpoint)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"code":500,"message":"Something bad happened."}`+"\n", w.Body.String())
}

func TestNewInternalError(t *testing.T) {
	want := &APIError{
		Code:    http.StatusInternalServerError,
		Message: "Internal Error",
	}
	got := NewInternalError(nil)

	assert.Equal(t, want, got)
}
