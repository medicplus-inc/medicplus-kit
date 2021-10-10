package encoding

import (
	"context"
	"encoding/json"
	"net/http"

	gokitHttp "github.com/go-kit/kit/transport/http"
)

func Encode() gokitHttp.EncodeResponseFunc {
	return func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if nil == response {
			w.WriteHeader(http.StatusNoContent)
			_ = json.NewEncoder(w).Encode(nil)
			return nil
		}

		_ = json.NewEncoder(w).Encode(response)

		return nil
	}
}
