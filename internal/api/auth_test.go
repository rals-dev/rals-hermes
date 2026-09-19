package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const bffKey = "correct-horse-battery-staple"

func authedDeps(t *testing.T) Deps {
	t.Helper()
	return newTestDeps(t, time.Second, profileSpec{"default", serveFixture(t, "default/health_detailed.json")})
}

func login(t *testing.T, h http.Handler, key string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"key": key})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/session", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "100.64.0.9:5555"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			return c
		}
	}
	t.Fatalf("no %s cookie in response", sessionCookieName)
	return nil
}

func getWithCookie(h http.Handler, path string, c *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if c != nil {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAuth_APIRequiresSessionCookie(t *testing.T) {
	h := NewHandler(authedDeps(t))
	for _, path := range []string{"/api/overview", "/api/agents"} {
		rec := getWithCookie(h, path, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without cookie: status = %d, want 401", path, rec.Code)
		}
		var body errorBody
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body.Code != "unauthorized" {
			t.Errorf("%s: code = %q, want unauthorized", path, body.Code)
		}
	}
	if rec := getWithCookie(h, "/healthz", nil); rec.Code != http.StatusOK {
		t.Errorf("/healthz must stay unauthenticated, got %d", rec.Code)
	}
}

func TestAuth_LoginWithWrongKeyIsRejected(t *testing.T) {
	h := NewHandler(authedDeps(t))
	rec := login(t, h, "wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("no cookie must be set on a failed login")
	}
}

func TestAuth_LoginSetsHttpOnlyStrictCookieThatGrantsAccess(t *testing.T) {
	h := NewHandler(authedDeps(t))
	rec := login(t, h, bffKey)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", rec.Code, rec.Body)
	}
	c := sessionCookie(t, rec)
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" {
		t.Errorf("cookie flags = HttpOnly:%v SameSite:%v Path:%q", c.HttpOnly, c.SameSite, c.Path)
	}
	if c.Secure {
		t.Error("cookie must not be Secure: Traefik terminates plain HTTP inside WireGuard (ADR-010)")
	}
	if c.MaxAge != int(time.Hour/time.Second) {
		t.Errorf("Max-Age = %d, want %d", c.MaxAge, int(time.Hour/time.Second))
	}
	if len(c.Value) < 32 || strings.Contains(c.Value, bffKey) {
		t.Errorf("cookie value %q must be a random token, never the key", c.Value)
	}
	if got := getWithCookie(h, "/api/overview", c); got.Code != http.StatusOK {
		t.Errorf("with cookie: status = %d, want 200: %s", got.Code, got.Body)
	}
}

func TestAuth_ForgedCookieIsRejected(t *testing.T) {
	h := NewHandler(authedDeps(t))
	forged := &http.Cookie{Name: sessionCookieName, Value: strings.Repeat("a", 64)}
	if rec := getWithCookie(h, "/api/overview", forged); rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuth_SessionExpiresAfterTTL(t *testing.T) {
	d := authedDeps(t)
	d.SessionTTL = time.Minute
	now := time.Unix(2_000_000, 0)
	d.now = func() time.Time { return now }
	h := NewHandler(d)

	c := sessionCookie(t, login(t, h, bffKey))
	now = now.Add(59 * time.Second)
	if rec := getWithCookie(h, "/api/overview", c); rec.Code != http.StatusOK {
		t.Errorf("before expiry: %d", rec.Code)
	}
	now = now.Add(2 * time.Second)
	if rec := getWithCookie(h, "/api/overview", c); rec.Code != http.StatusUnauthorized {
		t.Errorf("after expiry: status = %d, want 401", rec.Code)
	}
}

func TestAuth_LogoutRevokesSession(t *testing.T) {
	h := NewHandler(authedDeps(t))
	c := sessionCookie(t, login(t, h, bffKey))

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/session", nil)
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", rec.Code)
	}
	cleared := sessionCookie(t, rec)
	if cleared.MaxAge >= 0 && cleared.Value != "" {
		t.Errorf("logout must clear the cookie, got MaxAge=%d Value=%q", cleared.MaxAge, cleared.Value)
	}
	if got := getWithCookie(h, "/api/overview", c); got.Code != http.StatusUnauthorized {
		t.Errorf("after logout: status = %d, want 401", got.Code)
	}
}

func TestAuth_RepeatedFailuresAreRateLimited(t *testing.T) {
	h := NewHandler(authedDeps(t))
	var last *httptest.ResponseRecorder
	for range loginBurst + 1 {
		last = login(t, h, "wrong")
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("after %d failures: status = %d, want 429", loginBurst+1, last.Code)
	}
	var body errorBody
	_ = json.Unmarshal(last.Body.Bytes(), &body)
	if body.Code != "too_many_attempts" {
		t.Errorf("code = %q", body.Code)
	}
	// Even the correct key is refused while limited — no oracle for guessing.
	if rec := login(t, h, bffKey); rec.Code != http.StatusTooManyRequests {
		t.Errorf("correct key while limited: status = %d, want 429", rec.Code)
	}
}

func TestAuth_MalformedLoginBodyIsBadRequest(t *testing.T) {
	h := NewHandler(authedDeps(t))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/session", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
