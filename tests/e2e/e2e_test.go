package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// E2E tests hit a real running server.
// Start the stack with `make docker-up && make seed` before running.
// Run with: go test ./tests/e2e/... -v -tags e2e

const defaultBaseURL = "http://localhost:3000"

func baseURL() string {
	if u := os.Getenv("E2E_BASE_URL"); u != "" {
		return u
	}
	return defaultBaseURL
}

// ── HTTP client helpers ───────────────────────────────────────────────────────

type apiClient struct {
	base   string
	client *http.Client
	token  string
}

func newClient() *apiClient {
	return &apiClient{
		base:   baseURL(),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *apiClient) withToken(token string) *apiClient {
	cp := *c
	cp.token = token
	return &cp
}

func (c *apiClient) do(method, path string, body interface{}, headers map[string]string) (*http.Response, map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, c.base+path, reqBody)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return resp, result, nil
}

func (c *apiClient) post(path string, body interface{}) (*http.Response, map[string]interface{}, error) {
	return c.do(http.MethodPost, path, body, nil)
}

func (c *apiClient) get(path string) (*http.Response, map[string]interface{}, error) {
	return c.do(http.MethodGet, path, nil, nil)
}

// ── Helper ────────────────────────────────────────────────────────────────────

func skipIfNoServer(t *testing.T) {
	t.Helper()
	resp, err := http.Get(baseURL() + "/health/live")
	if err != nil || resp.StatusCode != 200 {
		t.Skipf("E2E server not running at %s — skipping", baseURL())
	}
}

// ── Test cases ────────────────────────────────────────────────────────────────

func TestE2E_HealthCheck(t *testing.T) {
	skipIfNoServer(t)

	resp, body, err := newClient().get("/health")
	if err != nil {
		t.Fatalf("health check: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	if body["status"] == "healthy" || body["status"] == "degraded" {
		t.Logf("health status: %v", body["status"])
	}
}

func TestE2E_Login_ValidCredentials(t *testing.T) {
	skipIfNoServer(t)

	client := newClient()
	resp, body, err := client.post("/v1/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "Admin@1234",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %v", resp.StatusCode, body)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok || data["accessToken"] == "" {
		t.Error("expected accessToken in response data")
	}
	t.Logf("login OK — token prefix: %s...", fmt.Sprintf("%v", data["accessToken"])[:20])
}

func TestE2E_Login_InvalidCredentials(t *testing.T) {
	skipIfNoServer(t)

	resp, _, _ := newClient().post("/v1/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "WrongPassword",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", resp.StatusCode)
	}
}

func TestE2E_AuthFlow_LoginRefreshLogout(t *testing.T) {
	skipIfNoServer(t)

	client := newClient()

	// Login
	resp, body, err := client.post("/v1/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "User@1234",
	})
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("login failed: %v / %d", err, resp.StatusCode)
	}
	data := body["data"].(map[string]interface{})
	accessToken := data["accessToken"].(string)
	refreshToken := data["refreshToken"].(string)

	// Access protected route
	authedClient := client.withToken(accessToken)
	resp, meBody, _ := authedClient.get("/v1/me")
	if resp.StatusCode != 200 {
		t.Errorf("GET /v1/me: want 200, got %d", resp.StatusCode)
	}
	t.Logf("user: %v", meBody["data"])

	// Refresh
	resp, refreshBody, _ := client.post("/v1/auth/refresh", map[string]string{
		"refreshToken": refreshToken,
	})
	if resp.StatusCode != 200 {
		t.Errorf("refresh: want 200, got %d", resp.StatusCode)
	}
	newAccessToken := refreshBody["data"].(map[string]interface{})["accessToken"].(string)
	if newAccessToken == accessToken {
		t.Error("expected new access token after refresh")
	}

	// Logout
	loggedOutClient := client.withToken(newAccessToken)
	resp, _, _ = loggedOutClient.do(http.MethodDelete, "/v1/auth/logout", nil, nil)
	if resp.StatusCode != 200 {
		t.Errorf("logout: want 200, got %d", resp.StatusCode)
	}

	// Old token should now be rejected
	resp, _, _ = authedClient.get("/v1/me")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("revoked token: want 401, got %d", resp.StatusCode)
	}
}

func TestE2E_AppVersionCheck_iOS(t *testing.T) {
	skipIfNoServer(t)

	resp, body, _ := newClient().do(http.MethodGet, "/v1/app-version/check", nil, map[string]string{
		"X-App-Version":  "1.0.0",
		"X-App-Platform": "ios",
	})
	if resp.StatusCode != 200 {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	data := body["data"].(map[string]interface{})
	t.Logf("version status: %v (current: %v)", data["status"], data["currentVersion"])
}

func TestE2E_RateLimit(t *testing.T) {
	skipIfNoServer(t)

	client := newClient()
	var lastStatus int

	// Fire many requests rapidly to trigger rate limit
	for i := 0; i < 120; i++ {
		resp, _, _ := client.post("/v1/auth/login", map[string]string{
			"email":    fmt.Sprintf("fake%d@example.com", i),
			"password": "wrong",
		})
		lastStatus = resp.StatusCode
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Logf("rate limit triggered after %d requests", i+1)
			return
		}
	}
	t.Logf("last status after 120 requests: %d", lastStatus)
}

func TestE2E_SecurityHeaders(t *testing.T) {
	skipIfNoServer(t)

	resp, _, _ := newClient().get("/health/live")
	if resp == nil {
		t.Skip("server not reachable")
	}

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	}
	for header, expected := range checks {
		got := resp.Header.Get(header)
		if got != expected {
			t.Errorf("header %q: want %q, got %q", header, expected, got)
		}
	}
	t.Log("security headers OK")
}

func TestE2E_I18n_Indonesian(t *testing.T) {
	skipIfNoServer(t)

	resp, body, _ := newClient().do(
		http.MethodPost,
		"/v1/auth/login",
		map[string]string{"email": "admin@example.com", "password": "Admin@1234"},
		map[string]string{"x-custom-lang": "id"},
	)
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	msg, _ := body["message"].(string)
	if msg != "Login berhasil" {
		t.Errorf("expected Indonesian message, got %q", msg)
	}
}

func TestE2E_Notification_Preferences(t *testing.T) {
	skipIfNoServer(t)

	client := newClient()
	// Login first
	_, body, _ := client.post("/v1/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "User@1234",
	})
	token := body["data"].(map[string]interface{})["accessToken"].(string)
	authed := client.withToken(token)

	// Get preferences
	resp, prefsBody, _ := authed.get("/v1/me/notifications/preferences")
	if resp.StatusCode != 200 {
		t.Fatalf("get prefs: want 200, got %d", resp.StatusCode)
	}
	t.Logf("preferences: %v", prefsBody["data"])

	// Update preferences
	resp, _, _ = authed.do(http.MethodPatch, "/v1/me/notifications/preferences",
		map[string]interface{}{
			"channels": map[string]bool{"email": false},
		}, nil)
	if resp.StatusCode != 200 {
		t.Errorf("update prefs: want 200, got %d", resp.StatusCode)
	}
}
