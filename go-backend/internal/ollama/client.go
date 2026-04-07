package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-backend/internal/logger"
	"go-backend/shared"
)

// Client communicates with a llama.cpp llama-server OpenAI-compatible API.
type Client struct {
	baseURL string
	client  *http.Client
	logger  *logger.Logger
}

// NewClient creates a new llama.cpp client.
func NewClient(baseURL string, log *logger.Logger) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		logger: log,
	}
}

// GenerateCompletion generates text completion using an OpenAI-compatible chat API.
func (c *Client) GenerateCompletion(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	c.logger.Debug("llama.cpp completion", "model", model, "system_prompt_len", len(systemPrompt), "prompt_len", len(userPrompt))

	messages := make([]shared.OpenAIChatMessage, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, shared.OpenAIChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, shared.OpenAIChatMessage{Role: "user", Content: userPrompt})

	req := shared.OpenAIChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Stream:      false,
		Temperature: 0.3,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		c.logger.Error("llama.cpp request failed", "error", err.Error())
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("llama.cpp api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
		c.logger.Error("llama.cpp error", "status", resp.StatusCode, "body", strings.TrimSpace(string(bodyBytes)))
		return "", err
	}

	var result shared.OpenAIChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("llama.cpp response had no choices")
	}

	response := strings.TrimSpace(result.Choices[0].Message.Content)
	c.logger.Debug("llama.cpp completion done", "response_len", len(response))
	return response, nil
}

// Health checks if llama.cpp is available.
func (c *Client) Health(ctx context.Context) (bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return false, err
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
