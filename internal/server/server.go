// Package server implements the token-gated local HTTP server that serves
// the embedded web UI and the JSON API for valve-node.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

// cookieName is the name of the cookie that carries the session token once
// it has been established via the ?token= query parameter.
const cookieName = "valve_node_token"

// Config configures a Server.
type Config struct {
	// Bind is the host:port the server listens on, e.g. "127.0.0.1:8799".
	Bind string
	// Token is the session token that authorizes API and UI requests.
	Token string
	// UI is the filesystem the static web UI is served from.
	UI fs.FS
}

// Server is the valve-node local HTTP server.
type Server struct {
	cfg Config
}

// New constructs a Server from the given Config.
func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// NewSessionToken returns a new random session token: 16 bytes of
// crypto/rand, hex-encoded to 32 characters.
func NewSessionToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// Handler builds the server's http.Handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}` + "\n"))
	})

	uiHandler := http.FileServerFS(s.cfg.UI)
	mux.Handle("/", uiHandler)

	return s.authMiddleware(mux)
}

// authMiddleware enforces the session token on every request. The token may
// arrive as an Authorization: Bearer header, a valve_node_token cookie, or a
// ?token= query parameter. A valid ?token= query parameter sets the cookie
// and redirects to the same path without the query parameter.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("token"); q != "" {
			if q != s.cfg.Token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    q,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			http.Redirect(w, r, r.URL.Path, http.StatusFound)
			return
		}

		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if tok, ok := strings.CutPrefix(authHeader, "Bearer "); ok && tok == s.cfg.Token {
				next.ServeHTTP(w, r)
				return
			}
		}

		if c, err := r.Cookie(cookieName); err == nil && c.Value == s.cfg.Token {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// ListenAndServe runs the server until ctx is canceled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:    s.cfg.Bind,
		Handler: s.Handler(),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return httpServer.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// cookiejarNew is a small helper wrapping net/http/cookiejar.New(nil), kept
// here so tests can create a jar without importing cookiejar directly.
func cookiejarNew() (*cookiejar.Jar, error) {
	return cookiejar.New(nil)
}
