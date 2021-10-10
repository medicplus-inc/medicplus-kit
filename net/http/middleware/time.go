package middleware

import (
	"context"
	"net/http"
	"time"
)

// noinspection ALL
const (
	KEY_REQUEST_TIME = "Hbm:T37.[ewrN;Ns"
)

func RequestTime(next http.Handler)  http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, KEY_REQUEST_TIME, time.Now())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
