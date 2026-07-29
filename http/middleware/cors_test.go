package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/config"
	"github.com/ymg2006/rustdesk-api/v2/global"
)

// setupCorsTestRouter builds a gin engine with the Cors middleware and registers a test handler that returns 200.
func setupCorsTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Cors())
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	return r
}

// withAllowOrigins sets the global CORS allowlist in tests to avoid zero-value panics or undefined behavior.
func withAllowOrigins(origins ...string) {
	global.Config = config.Config{
		Cors: config.Cors{AllowOrigins: origins},
	}
}

// TestCors_EmptyWhitelist_BlocksAnyOrigin
// An empty allowlist must not reflect any Origin or send credential headers; same-origin/cross-origin requests still reach the handler.
func TestCors_EmptyWhitelist_BlocksAnyOrigin(t *testing.T) {
	withAllowOrigins() // empty allowlist
	r := setupCorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected handler to return 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO should not be set when allowlist is empty, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("credential header should not be set when allowlist is empty, got %q", got)
	}
}

// TestCors_AllowedOrigin_ReflectsAndAllowsCredentials
// An allowlisted Origin should be reflected exactly and must include Access-Control-Allow-Credentials: true.
func TestCors_AllowedOrigin_ReflectsAndAllowsCredentials(t *testing.T) {
	withAllowOrigins("https://a.example.com")
	r := setupCorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://a.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected handler to return 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://a.example.com" {
		t.Errorf("expected ACAO to reflect https://a.example.com, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected credential header to be true, got %q", got)
	}
}

// TestCors_DisallowedOrigin_NoCorsHeaders
// A non-allowlisted Origin should not be reflected and should not allow credentials.
func TestCors_DisallowedOrigin_NoCorsHeaders(t *testing.T) {
	withAllowOrigins("https://a.example.com")
	r := setupCorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://b.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected handler to return 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO should not be set for non-allowlisted origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("credential header should not be set for non-allowlisted origin, got %q", got)
	}
}

// TestCors_Preflight_AllowedOrigin_Returns204
// An allowlisted OPTIONS preflight should return 204 directly and include ACAO.
func TestCors_Preflight_AllowedOrigin_Returns204(t *testing.T) {
	withAllowOrigins("https://a.example.com")
	r := setupCorsTestRouter()

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://a.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("allowlisted preflight should return 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://a.example.com" {
		t.Errorf("preflight response should include ACAO, got %q", got)
	}
}

// TestCors_Preflight_DisallowedOrigin_Not204
// A non-allowlisted OPTIONS preflight should not return 204 and should not set ACAO.
func TestCors_Preflight_DisallowedOrigin_Not204(t *testing.T) {
	withAllowOrigins("https://a.example.com")
	r := setupCorsTestRouter()

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://b.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusNoContent {
		t.Fatalf("non-allowlisted preflight should not return 204")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("non-allowlisted preflight should not set ACAO, got %q", got)
	}
}

// TestCors_NoOriginHeader_ReachesHandler
// Same-origin requests without an Origin header should reach the handler and should not set extra CORS headers.
func TestCors_NoOriginHeader_ReachesHandler(t *testing.T) {
	withAllowOrigins("https://a.example.com")
	r := setupCorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil) // no Origin header
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected same-origin request to return 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO should not be set without Origin, got %q", got)
	}
}

// TestCors_CaseInsensitiveMatch
// Allowlist matching is case-insensitive to keep behavior stable and prevent case-based bypasses.
func TestCors_CaseInsensitiveMatch(t *testing.T) {
	withAllowOrigins("https://a.example.com")
	r := setupCorsTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "HTTPS://A.EXAMPLE.COM")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected handler to return 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "HTTPS://A.EXAMPLE.COM" {
		t.Errorf("after case-insensitive match, ACAO should reflect original Origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected credential header to be true, got %q", got)
	}
}
