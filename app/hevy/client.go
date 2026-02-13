package hevy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL         = "https://api.hevyapp.com/v1"
	defaultPageSize = 10
	pageDelay       = 200 * time.Millisecond
	maxRetries      = 3
)

var (
	ErrUnauthorized = errors.New("hevy: unauthorized (invalid API key)")
	ErrRateLimited  = errors.New("hevy: rate limited")
	ErrNotFound     = errors.New("hevy: not found")
)

// Client is an HTTP client for the Hevy API.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Hevy API client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("hevy: failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := baseURL + path

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("hevy: failed to create request: %w", err)
		}
		req.Header.Set("api-key", c.apiKey)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("hevy: request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("hevy: failed to read response: %w", err)
			continue
		}

		switch resp.StatusCode {
		case http.StatusOK, http.StatusCreated:
			return respBody, nil
		case http.StatusUnauthorized:
			return nil, ErrUnauthorized
		case http.StatusNotFound:
			return nil, ErrNotFound
		case http.StatusTooManyRequests:
			lastErr = ErrRateLimited
			continue
		case http.StatusBadRequest:
			var errResp ErrorResponse
			if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
				return nil, fmt.Errorf("hevy: bad request: %s", errResp.Error)
			}
			return nil, fmt.Errorf("hevy: bad request: %s", string(respBody))
		default:
			lastErr = fmt.Errorf("hevy: unexpected status %d: %s", resp.StatusCode, string(respBody))
			continue
		}
	}

	return nil, fmt.Errorf("hevy: max retries exceeded: %w", lastErr)
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

func (c *Client) post(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

func (c *Client) put(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPut, path, body)
}
