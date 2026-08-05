package handler

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"
)

// apiToken returns the bearer token from the environment. An empty value means
// no token is configured, in which case no request can authenticate (the guard
// fails closed).
func apiToken() string {
	return os.Getenv("UPTIME_API_TOKEN")
}

// tokenMatches reports whether the request carries a valid bearer token. When
// UPTIME_API_TOKEN is unset, no token is valid.
func tokenMatches(r *http.Request) bool {
	token := apiToken()
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(auth, prefix)), []byte(token)) == 1
}

func rejectUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="uptime"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

// RequireAPIToken guards a handler with a bearer token read from the
// UPTIME_API_TOKEN environment variable. Requests without a matching
// `Authorization: Bearer <token>` header receive 401 Unauthorized.
//
// If UPTIME_API_TOKEN is not set, all requests are denied (fail closed).
func RequireAPIToken(next http.HandlerFunc) http.HandlerFunc {
	if apiToken() == "" {
		log.Printf("[auth] UPTIME_API_TOKEN not set; all mutating requests will be denied")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if !tokenMatches(r) {
			rejectUnauthorized(w)
			return
		}
		next(w, r)
	}
}
