package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func testServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
	}})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func TestHealthRequiresToken(t *testing.T) {
	ts, token := testServer(t)
	res, _ := http.Get(ts.URL + "/api/health")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401", res.StatusCode)
	}
	req, _ := http.NewRequest("GET", ts.URL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, _ = http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("with token: got %d, want 200", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != `{"ok":true}`+"\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestTokenQueryParamSetsCookieAndServesUI(t *testing.T) {
	ts, token := testServer(t)
	jar, _ := cookiejarNew()
	client := &http.Client{Jar: jar}
	res, _ := client.Get(ts.URL + "/?token=" + token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("ui with token: %d", res.StatusCode)
	}
	// Cookie now authorizes the API without the header.
	res, _ = client.Get(ts.URL + "/api/health")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("api via cookie: %d", res.StatusCode)
	}
}

// TestWrongTokenRejected_HeaderAndCookie locks in that the constant-time
// token comparison (crypto/subtle.ConstantTimeCompare) still enforces exact
// equality on both the Authorization header and cookie auth paths — wrong
// tokens (including different-length ones, which ConstantTimeCompare
// short-circuits on) must be rejected exactly as with the old `==` compare.
func TestWrongTokenRejected_HeaderAndCookie(t *testing.T) {
	ts, token := testServer(t)

	req, _ := http.NewRequest("GET", ts.URL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer wrong-token-wrong-token-wrong")
	res, _ := http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong bearer token: got %d, want 401", res.StatusCode)
	}

	req, _ = http.NewRequest("GET", ts.URL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer short")
	res, _ = http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("shorter-than-token bearer token: got %d, want 401", res.StatusCode)
	}

	req, _ = http.NewRequest("GET", ts.URL+"/api/health", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "wrong-cookie-wrong-cookie-value"})
	res, _ = http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong cookie token: got %d, want 401", res.StatusCode)
	}

	// Sanity: the real token still works on both paths.
	req, _ = http.NewRequest("GET", ts.URL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, _ = http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("correct bearer token: got %d, want 200", res.StatusCode)
	}
	req, _ = http.NewRequest("GET", ts.URL+"/api/health", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	res, _ = http.DefaultClient.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("correct cookie token: got %d, want 200", res.StatusCode)
	}
}

func TestNewSessionTokenIsRandomHex(t *testing.T) {
	a, b := NewSessionToken(), NewSessionToken()
	if len(a) != 32 || a == b {
		t.Fatalf("tokens: %q %q", a, b)
	}
}
