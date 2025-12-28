package middleware

import (
	"net/http"
)

// CacheMiddleware sets the Cache-Control header based on the version.
func CacheHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Cache-Control", "public, max-age=31536000")

		next.ServeHTTP(w, r)
	})
}
