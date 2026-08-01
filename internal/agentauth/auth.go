package agentauth

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

// Token returns the shared agent auth token from the environment.
// When empty, agent endpoints accept unauthenticated requests (dev only).
func Token() string {
	return os.Getenv("STRATABENCH_AGENT_TOKEN")
}

// Middleware protects agent HTTP handlers when STRATABENCH_AGENT_TOKEN is set.
func Middleware(next http.Handler) http.Handler {
	token := Token()
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !Authorized(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Authorized checks Bearer or X-StrataBench-Token against the expected token.
func Authorized(r *http.Request, expected string) bool {
	if expected == "" {
		return true
	}
	got := r.Header.Get("Authorization")
	if strings.HasPrefix(got, "Bearer ") {
		got = strings.TrimSpace(strings.TrimPrefix(got, "Bearer "))
	}
	if got == "" {
		got = r.Header.Get("X-StrataBench-Token")
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

// SetAuthHeader attaches the coordinator token to outbound agent requests.
func SetAuthHeader(req *http.Request) {
	if t := Token(); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
}
