package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ollamaDefaultBase is Ollama's default local listener, used when New's
// baseURL is "".
const ollamaDefaultBase = "http://localhost:11434"

const ollamaModel = "llama3.2"

type ollamaProvider struct {
	baseURL string
	client  *http.Client
}

func newOllamaProvider(baseURL string) *ollamaProvider {
	if baseURL == "" {
		baseURL = ollamaDefaultBase
	}
	return &ollamaProvider{baseURL: baseURL, client: newHTTPClient()}
}

func (p *ollamaProvider) Name() string { return "ollama" }

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ollamaChatResponse struct {
	Message openAIChatMessage `json:"message"`
	Done    bool              `json:"done"`
}

func (p *ollamaProvider) Explain(ctx context.Context, req ExplainRequest) (string, error) {
	prompt := buildPrompt(req)

	reqBody := ollamaChatRequest{
		Model:    ollamaModel,
		Messages: []openAIChatMessage{{Role: "user", Content: prompt}},
		// Ollama's /api/chat streams NDJSON by default; explicitly opt out
		// so the response is a single JSON object.
		Stream: false,
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ollama: encode request: %w", err)
	}

	endpoint := p.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("ollama: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama: request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("ollama: read response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: unexpected status %d: %s", res.StatusCode, string(respBody))
	}

	var parsed ollamaChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("ollama: decode response: %w", err)
	}
	if parsed.Message.Content == "" {
		return "", fmt.Errorf("ollama: response had no message content")
	}
	return parsed.Message.Content, nil
}
