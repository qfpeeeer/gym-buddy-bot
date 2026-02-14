package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	openaiBaseURL  = "https://api.openai.com/v1/chat/completions"
	defaultModel   = "gpt-4.1-mini"
	requestTimeout = 60 * time.Second
)

// Client is an OpenAI chat completions client.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient creates a new OpenAI client.
func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = defaultModel
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// IsConfigured returns true if the client has an API key.
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat sends a system prompt and user message to OpenAI and returns the response.
func (c *Client) Chat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiBaseURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var chatResp chatResponse
		if json.Unmarshal(body, &chatResp) == nil && chatResp.Error != nil {
			return "", fmt.Errorf("OpenAI error: %s", chatResp.Error.Message)
		}
		return "", fmt.Errorf("OpenAI error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return chatResp.Choices[0].Message.Content, nil
}
