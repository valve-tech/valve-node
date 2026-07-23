package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// groqDefaultBase is Groq's real API host, used when New's baseURL is "".
const groqDefaultBase = "https://api.groq.com"

const groqModel = "llama-3.3-70b-versatile"

type groqProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func newGroqProvider(apiKey, baseURL string) *groqProvider {
	if baseURL == "" {
		baseURL = groqDefaultBase
	}
	return &groqProvider{apiKey: apiKey, baseURL: baseURL, client: newHTTPClient()}
}

func (p *groqProvider) Name() string { return "groq" }

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
}

func (p *groqProvider) Explain(ctx context.Context, req ExplainRequest) (string, error) {
	prompt := buildPrompt(req)

	reqBody := openAIChatRequest{
		Model:    groqModel,
		Messages: []openAIChatMessage{{Role: "user", Content: prompt}},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("groq: encode request: %w", err)
	}

	endpoint := p.baseURL + "/openai/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("groq: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	res, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("groq: request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("groq: read response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq: unexpected status %d: %s", res.StatusCode, string(respBody))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("groq: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("groq: response had no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
