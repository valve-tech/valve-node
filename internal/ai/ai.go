// Package ai wraps a handful of LLM providers behind one Provider
// interface so logwatch (or an operator) can ask "explain these log
// lines" and get back a plain-English diagnosis regardless of which
// backend is configured.
package ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ExplainRequest is the context handed to a Provider for one explain call.
type ExplainRequest struct {
	ChainName, ExecClient, BeaconClient string
	Syncing                             bool
	Lines                               []string // capped at 80 lines / 8KB by Explain callers
}

// Provider is an LLM backend that can explain a batch of log lines.
type Provider interface {
	Name() string
	Explain(ctx context.Context, req ExplainRequest) (string, error)
}

// httpTimeout bounds every provider's HTTP call.
const httpTimeout = 30 * time.Second

// New returns a Provider by id. baseURL "" selects the provider's real
// default endpoint; tests pass an httptest server URL instead.
func New(id, apiKey, baseURL string) (Provider, error) {
	switch id {
	case "gemini":
		return newGeminiProvider(apiKey, baseURL), nil
	case "groq":
		return newGroqProvider(apiKey, baseURL), nil
	case "ollama":
		return newOllamaProvider(baseURL), nil
	default:
		return nil, fmt.Errorf("ai: unknown provider %q", id)
	}
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: httpTimeout}
}

// maxExplainLines and maxExplainBytes are a defensive second cap on top of
// the one ExplainRequest.Lines documents as the caller's job: callers
// (logwatch) are expected to already cap to 80 lines / 8KB before calling
// Explain, but the cap is cheap and the cost of a caller forgetting (an
// oversized provider request, possibly rejected or expensive) is worse
// than re-checking it here.
const (
	maxExplainLines = 80
	maxExplainBytes = 8 * 1024
)

// capLines trims lines to at most maxExplainLines entries and
// maxExplainBytes total bytes, keeping the most recent (tail) lines —
// on the assumption that whatever just happened is most relevant to an
// operator asking "what's wrong right now".
func capLines(lines []string) []string {
	if len(lines) > maxExplainLines {
		lines = lines[len(lines)-maxExplainLines:]
	}
	total := 0
	for _, l := range lines {
		total += len(l) + 1
	}
	for total > maxExplainBytes && len(lines) > 0 {
		total -= len(lines[0]) + 1
		lines = lines[1:]
	}
	return lines
}

// buildPrompt renders the one shared prompt template used by every
// provider: a preamble describing the node's context, followed by the
// (capped) log lines.
func buildPrompt(req ExplainRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b,
		"You are diagnosing a %s node running %s+%s; syncing=%v. "+
			"Explain these log lines for an operator in plain English, "+
			"most-likely cause first, then the fix. Keep it under 150 words.\n\n",
		req.ChainName, req.ExecClient, req.BeaconClient, req.Syncing,
	)
	for _, line := range capLines(req.Lines) {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
