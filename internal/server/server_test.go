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

func TestNewSessionTokenIsRandomHex(t *testing.T) {
	a, b := NewSessionToken(), NewSessionToken()
	if len(a) != 32 || a == b {
		t.Fatalf("tokens: %q %q", a, b)
	}
}
