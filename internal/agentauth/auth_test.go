package agentauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareNoTokenAllowsAll(t *testing.T) {
	t.Setenv("STRATABENCH_AGENT_TOKEN", "")
	called := false
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called || rr.Code != http.StatusOK {
		t.Fatalf("code=%d called=%v", rr.Code, called)
	}
}

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	t.Setenv("STRATABENCH_AGENT_TOKEN", "secret")
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestMiddlewareAcceptsBearer(t *testing.T) {
	t.Setenv("STRATABENCH_AGENT_TOKEN", "secret")
	called := false
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called || rr.Code != http.StatusOK {
		t.Fatalf("code=%d called=%v", rr.Code, called)
	}
}
