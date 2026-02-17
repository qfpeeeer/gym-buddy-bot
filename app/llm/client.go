package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	openaiBaseURL  = "https://api.openai.com/v1/chat/completions"
	defaultModel   = "gpt-4.1-mini"
	requestTimeout = 120 * time.Second
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

// isReasoningModel returns true for models that use reasoning tokens (gpt-5*, o1*, o3*, o4*).
func isReasoningModel(model string) bool {
	m := strings.ToLower(model)
	return strings.HasPrefix(m, "gpt-5") ||
		strings.HasPrefix(m, "o1") ||
		strings.HasPrefix(m, "o3") ||
		strings.HasPrefix(m, "o4")
}

// chatRequest for standard models (gpt-4.1 family).
type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

// chatRequestReasoning for reasoning models (gpt-5 family, o-series).
type chatRequestReasoning struct {
	Model               string        `json:"model"`
	Messages            []chatMessage `json:"messages"`
	MaxCompletionTokens int           `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string        `json:"reasoning_effort,omitempty"`
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
// maxTokens limits response length (0 = model default).
func (c *Client) Chat(ctx context.Context, systemPrompt, userMessage string, maxTokens int) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	var data []byte
	var err error

	if isReasoningModel(c.model) {
		req := chatRequestReasoning{
			Model:           c.model,
			Messages:        messages,
			ReasoningEffort: "low",
		}
		// For reasoning models, don't set a small token limit — reasoning tokens
		// consume the budget and can leave nothing for the actual response.
		// Only set if explicitly large enough (>= 8000).
		if maxTokens >= 8000 {
			req.MaxCompletionTokens = maxTokens
		}
		data, err = json.Marshal(req)
	} else {
		req := chatRequest{
			Model:     c.model,
			Messages:  messages,
			MaxTokens: maxTokens,
		}
		data, err = json.Marshal(req)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiBaseURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
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
		log.Printf("[warn] OpenAI returned 0 choices, body: %s", string(body))
		return "", fmt.Errorf("no response from OpenAI")
	}

	content := chatResp.Choices[0].Message.Content
	if content == "" {
		log.Printf("[warn] OpenAI returned empty content, body: %s", string(body))
		return "", fmt.Errorf("OpenAI returned an empty response (reasoning model may need higher token budget)")
	}

	return content, nil
}
