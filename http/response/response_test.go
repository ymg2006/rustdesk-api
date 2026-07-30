package response

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/ymg2006/rustdesk-api/v2/global"
)

// newTestContext builds a gin test context and redirects the global Logger to a readable buffer.
// This avoids nil Logger panics inside Fail and verifies that original >=500 messages are logged
// server-side instead of being echoed to clients.
func newTestContext() (*gin.Context, *httptest.ResponseRecorder, *bytes.Buffer) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	// gin v1.9.0 CreateTestContext returns (*Context, *Engine); the second value is not needed here.
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	var logBuf bytes.Buffer
	global.Logger = logrus.New()
	global.Logger.SetOutput(&logBuf)
	global.Logger.SetLevel(logrus.InfoLevel)
	return c, w, &logBuf
}

// assertGeneric500Body asserts that the response body uses a generic message and contains no sensitive details.
func assertGeneric500Body(t *testing.T, w *httptest.ResponseRecorder, logBuf *bytes.Buffer, originalMessage string) {
	t.Helper()

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (body=%s)", err, w.Body.String())
	}
	if body["code"] != float64(500) {
		t.Errorf("expected code=500, got %v", body["code"])
	}
	if body["message"] != "Internal server error." && body["message"] != "ServerInternalError" {
		t.Errorf("expected generic internal server error message, got %v", body["message"])
	}
	// The data field should keep the expected JSON shape and be serialized even when nil.
	if _, ok := body["data"]; !ok {
		t.Errorf("response is missing data field; expected shape {code,message,data}: %s", w.Body.String())
	}

	raw := w.Body.String()
	for _, leak := range []string{"Error 1146", "tb_x", "rustdesk_api", "database exploded"} {
		if strings.Contains(raw, leak) {
			t.Errorf("response body leaked sensitive detail %q: %s", leak, raw)
		}
	}

	// The original message should be logged only server-side and must not appear in the client response body.
	if logBuf != nil && originalMessage != "" {
		if !strings.Contains(logBuf.String(), originalMessage) {
			t.Errorf("original message was not logged server-side: log=%s", logBuf.String())
		}
		if strings.Contains(raw, originalMessage) {
			t.Errorf("original message should not be echoed to clients: %s", raw)
		}
	}
}

// TestFail_ServerError_500_SuppressesDetailAndReturnsGenericMessage
// code=500 must discard caller-provided details, return only a generic message,
// and avoid exposing SQL, table names, database names, or business text in the response body.
func TestFail_ServerError_500_SuppressesDetailAndReturnsGenericMessage(t *testing.T) {
	c, w, logBuf := newTestContext()

	detail := "database exploded: Error 1146: Table 'rustdesk_api.tb_x' doesn't exist"
	Fail(c, 500, detail)

	assertGeneric500Body(t, w, logBuf, detail)
}

// TestFail_ServerError_500_LogsOriginalMessageServerSide
// For code >= 500, the original message should be written to global.Logger.Error server-side.
func TestFail_ServerError_500_LogsOriginalMessageServerSide(t *testing.T) {
	c, _, logBuf := newTestContext()

	detail := "failed to read config file: open /etc/secret: permission denied"
	Fail(c, 500, detail)

	if !strings.Contains(logBuf.String(), detail) {
		t.Errorf(">=500 original message was not logged server-side: %s", logBuf.String())
	}
	if !strings.Contains(logBuf.String(), "server error response suppressed for client") {
		t.Errorf("log should indicate that the message was suppressed for clients: %s", logBuf.String())
	}
}

// TestFail_ClientError_400_PassesThroughMessage
// Negative case: for code < 500, the message must pass through unchanged so validation hints are not lost.
func TestFail_ClientError_400_PassesThroughMessage(t *testing.T) {
	c, w, _ := newTestContext()

	msg := "invalid parameter: name cannot be empty"
	Fail(c, 400, msg)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (body=%s)", err, w.Body.String())
	}
	if body["code"] != float64(400) {
		t.Errorf("expected code=400, got %v", body["code"])
	}
	if body["message"] != msg {
		t.Errorf("expected message to pass through unchanged as %q, got %v", msg, body["message"])
	}
}

// TestFail_ServerError_500_EmptyMessage_ReturnsGeneric
// Boundary case: a 500 with an empty caller-provided message should still return a generic message without panic or empty body.
func TestFail_ServerError_500_EmptyMessage_ReturnsGeneric(t *testing.T) {
	c, w, _ := newTestContext()

	Fail(c, 500, "")

	if w.Code == 0 {
		t.Fatalf("no response was produced; possible panic")
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (body=%s)", err, w.Body.String())
	}
	if body["message"] != "Internal server error." && body["message"] != "ServerInternalError" {
		t.Errorf("500 with empty message should return a generic message, got %v (body=%s)", body["message"], w.Body.String())
	}
	if body["message"] == "" {
		t.Errorf("500 response message should not be empty")
	}
}

// TestFail_Boundary_499PassesThrough_500Suppressed
// Boundary consistency: exactly 499 (<500) passes through; exactly 500 is suppressed.
func TestFail_Boundary_499PassesThrough_500Suppressed(t *testing.T) {
	// 499 passes through.
	c499, w499, _ := newTestContext()
	Fail(c499, 499, "custom business failure")
	var b499 map[string]interface{}
	_ = json.Unmarshal(w499.Body.Bytes(), &b499)
	if b499["message"] != "custom business failure" {
		t.Errorf("499 should pass through message, got %v", b499["message"])
	}

	// 500 is suppressed.
	c500, w500, _ := newTestContext()
	Fail(c500, 500, "internal error detail")
	var b500 map[string]interface{}
	_ = json.Unmarshal(w500.Body.Bytes(), &b500)
	if b500["message"] != "Internal server error." && b500["message"] != "ServerInternalError" {
		t.Errorf("500 should return generic message, got %v", b500["message"])
	}
}
