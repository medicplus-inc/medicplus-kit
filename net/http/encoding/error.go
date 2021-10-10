package encoding

import (
	"context"
	"encoding/json"
	"net/http"

	libError "github.com/medicplus-inc/medicplus-kit/error"
)

func EncodeError(ctx context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	code := http.StatusInternalServerError
	message := "Something Went Wrong"

	if sc, ok := err.(*libError.Error); ok {
		code = sc.StatusCode
		message = sc.Message
	}

	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   err.Error(),
		"code":    code,
		"message": message,
	})
}
