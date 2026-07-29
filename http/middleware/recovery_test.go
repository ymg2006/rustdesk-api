package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRecovery_ReturnsGeneric500OnPanic verifies that when a panic is recovered:
// 1) HTTP 200 is returned with business code 500, matching the site-wide 5xx envelope;
// 2) the response uses a generic message;
// 3) panic details, which may include SQL or stack traces, do not leak into the response body.
func TestRecovery_ReturnsGeneric500OnPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("boom: secret sql=SELECT * FROM users WHERE password='x'")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v (body=%s)", err, w.Body.String())
	}
	if body["code"] != float64(500) {
		t.Errorf("expected code 500, got %v", body["code"])
	}
	if body["message"] != "Internal server error." {
		t.Errorf("expected generic message, got %v", body["message"])
	}

	// Ensure panic details are not leaked into the response body.
	if strings.Contains(w.Body.String(), "boom") || strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "SELECT") {
		t.Errorf("panic details leaked to client: %s", w.Body.String())
	}
}

// TestRecovery_PassesThroughWhenNoPanic verifies normal pass-through when there is no panic.
func TestRecovery_PassesThroughWhenNoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
