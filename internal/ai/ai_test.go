package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testReq() ExplainRequest {
	return ExplainRequest{
		ChainName:    "PulseChain",
		ExecClient:   "reth",
		BeaconClient: "lighthouse",
		Syncing:      true,
		Lines: []string{
			"FATAL Fatal error: database is corrupt, please resync from a snapshot",
			"WARN Low peer count peers=1",
		},
	}
}

func assertPromptHasChainAndLines(t *testing.T, prompt string) {
	t.Helper()
	if !strings.Contains(prompt, "PulseChain") {
		t.Errorf("prompt missing chain name, got: %s", prompt)
	}
	if !strings.Contains(prompt, "reth") || !strings.Contains(prompt, "lighthouse") {
		t.Errorf("prompt missing client names, got: %s", prompt)
	}
	if !strings.Contains(prompt, "database is corrupt") || !strings.Contains(prompt, "Low peer count") {
		t.Errorf("prompt missing log lines, got: %s", prompt)
	}
}

// ---------------------------------------------------------------------
// New: provider selection
// ---------------------------------------------------------------------

func TestNewUnknownProviderErrors(t *testing.T) {
	if _, err := New("nope", "key", ""); err == nil {
		t.Fatal("New(\"nope\", ...) err = nil, want an error")
	}
}

func TestNewKnownProviders(t *testing.T) {
	for _, id := range []string{"gemini", "groq", "ollama"} {
		p, err := New(id, "key", "")
		if err != nil {
			t.Fatalf("New(%q, ...) err = %v, want nil", id, err)
		}
		if p.Name() != id {
			t.Errorf("New(%q, ...).Name() = %q, want %q", id, p.Name(), id)
		}
	}
}

// ---------------------------------------------------------------------
// gemini
// ---------------------------------------------------------------------

func TestGeminiExplainRequestShape(t *testing.T) {
	var gotPath, gotKey string
	var gotBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"it's a corrupt db, resync"}],"role":"model"}}]}`))
	}))
	defer ts.Close()

	p, err := New("gemini", "test-gemini-key", ts.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := p.Explain(context.Background(), testReq())
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if out != "it's a corrupt db, resync" {
		t.Errorf("Explain() = %q, want the canned text", out)
	}

	if gotPath != "/v1beta/models/gemini-2.0-flash:generateContent" {
		t.Errorf("path = %q, want /v1beta/models/gemini-2.0-flash:generateContent", gotPath)
	}
	if gotKey != "test-gemini-key" {
		t.Errorf("?key= = %q, want test-gemini-key", gotKey)
	}

	contents, _ := gotBody["contents"].([]any)
	if len(contents) == 0 {
		t.Fatalf("request body has no contents: %+v", gotBody)
	}
	first, _ := contents[0].(map[string]any)
	parts, _ := first["parts"].([]any)
	if len(parts) == 0 {
		t.Fatalf("request body contents[0] has no parts: %+v", first)
	}
	part, _ := parts[0].(map[string]any)
	prompt, _ := part["text"].(string)
	assertPromptHasChainAndLines(t, prompt)
}

func TestGeminiExplainNon200IsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"boom"}`))
	}))
	defer ts.Close()

	p, _ := New("gemini", "key", ts.URL)
	_, err := p.Explain(context.Background(), testReq())
	if err == nil {
		t.Fatal("Explain err = nil, want error on non-200")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("err = %v, want it to contain the status code 500", err)
	}
}

// ---------------------------------------------------------------------
// groq
// ---------------------------------------------------------------------

func TestGroqExplainRequestShape(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"low peers, open your P2P port"}}]}`))
	}))
	defer ts.Close()

	p, err := New("groq", "test-groq-key", ts.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := p.Explain(context.Background(), testReq())
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if out != "low peers, open your P2P port" {
		t.Errorf("Explain() = %q, want the canned text", out)
	}

	if gotPath != "/openai/v1/chat/completions" {
		t.Errorf("path = %q, want /openai/v1/chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-groq-key" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-groq-key")
	}
	if model, _ := gotBody["model"].(string); model != "llama-3.3-70b-versatile" {
		t.Errorf("model = %q, want llama-3.3-70b-versatile", model)
	}

	messages, _ := gotBody["messages"].([]any)
	if len(messages) == 0 {
		t.Fatalf("request body has no messages: %+v", gotBody)
	}
	last, _ := messages[len(messages)-1].(map[string]any)
	prompt, _ := last["content"].(string)
	assertPromptHasChainAndLines(t, prompt)
}

func TestGroqExplainNon200IsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer ts.Close()

	p, _ := New("groq", "key", ts.URL)
	_, err := p.Explain(context.Background(), testReq())
	if err == nil {
		t.Fatal("Explain err = nil, want error on non-200")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want it to contain the status code 401", err)
	}
}

func TestGroqExplainEmptyContentIsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":""}}]}`))
	}))
	defer ts.Close()

	p, _ := New("groq", "key", ts.URL)
	_, err := p.Explain(context.Background(), testReq())
	if err == nil {
		t.Fatal("Explain err = nil, want error on empty message content (match ollama's strictness)")
	}
}

// ---------------------------------------------------------------------
// ollama
// ---------------------------------------------------------------------

func TestOllamaExplainRequestShape(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model":"llama3.2","message":{"role":"assistant","content":"corrupt db, restore from snapshot"},"done":true}`))
	}))
	defer ts.Close()

	p, err := New("ollama", "", ts.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := p.Explain(context.Background(), testReq())
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if out != "corrupt db, restore from snapshot" {
		t.Errorf("Explain() = %q, want the canned text", out)
	}

	if gotPath != "/api/chat" {
		t.Errorf("path = %q, want /api/chat", gotPath)
	}
	if model, _ := gotBody["model"].(string); model != "llama3.2" {
		t.Errorf("model = %q, want llama3.2", model)
	}
	if stream, ok := gotBody["stream"].(bool); !ok || stream {
		t.Errorf("stream = %v (ok=%v), want false (non-streaming response)", gotBody["stream"], ok)
	}

	messages, _ := gotBody["messages"].([]any)
	if len(messages) == 0 {
		t.Fatalf("request body has no messages: %+v", gotBody)
	}
	last, _ := messages[len(messages)-1].(map[string]any)
	prompt, _ := last["content"].(string)
	assertPromptHasChainAndLines(t, prompt)
}

func TestOllamaExplainNon200IsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"model not loaded"}`))
	}))
	defer ts.Close()

	p, _ := New("ollama", "", ts.URL)
	_, err := p.Explain(context.Background(), testReq())
	if err == nil {
		t.Fatal("Explain err = nil, want error on non-200")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("err = %v, want it to contain the status code 503", err)
	}
}

// ---------------------------------------------------------------------
// prompt line capping (defensive; primary cap is the caller's job)
// ---------------------------------------------------------------------

func TestCapLinesCapsCountAndBytes(t *testing.T) {
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = strings.Repeat("x", 100) // 100 bytes/line, 200 lines = 20KB
	}
	capped := capLines(lines)
	if len(capped) > maxExplainLines {
		t.Errorf("capLines returned %d lines, want <= %d", len(capped), maxExplainLines)
	}
	total := 0
	for _, l := range capped {
		total += len(l) + 1
	}
	if total > maxExplainBytes {
		t.Errorf("capLines returned %d bytes, want <= %d", total, maxExplainBytes)
	}
}
