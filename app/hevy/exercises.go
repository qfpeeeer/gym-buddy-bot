package hevy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetExerciseTemplates fetches a page of exercise templates.
func (c *Client) GetExerciseTemplates(ctx context.Context, page, pageSize int) (*ExerciseTemplatesResponse, error) {
	data, err := c.get(ctx, fmt.Sprintf("/exercise_templates?page=%d&pageSize=%d", page, pageSize))
	if err != nil {
		return nil, err
	}
	var resp ExerciseTemplatesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode exercise templates: %w", err)
	}
	return &resp, nil
}

// GetAllExerciseTemplates fetches all exercise templates, paginating automatically.
func (c *Client) GetAllExerciseTemplates(ctx context.Context) ([]ExerciseTemplate, error) {
	var all []ExerciseTemplate
	page := 1

	for {
		resp, err := c.GetExerciseTemplates(ctx, page, defaultPageSize)
		if err != nil {
			return all, fmt.Errorf("hevy: failed to fetch exercise templates page %d: %w", page, err)
		}

		all = append(all, resp.ExerciseTemplates...)

		if page >= resp.PageCount {
			break
		}

		page++
		select {
		case <-ctx.Done():
			return all, ctx.Err()
		case <-time.After(pageDelay):
		}
	}

	return all, nil
}

// GetExerciseTemplateByID fetches a single exercise template by its ID.
func (c *Client) GetExerciseTemplateByID(ctx context.Context, id string) (*ExerciseTemplate, error) {
	data, err := c.get(ctx, "/exercise_templates/"+id)
	if err != nil {
		return nil, err
	}
	var t ExerciseTemplate
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode exercise template: %w", err)
	}
	return &t, nil
}

// CreateExerciseTemplate creates a custom exercise template.
// Returns the ID of the created template.
func (c *Client) CreateExerciseTemplate(ctx context.Context, req CreateExerciseRequest) (string, error) {
	data, err := c.post(ctx, "/exercise_templates", req)
	if err != nil {
		return "", err
	}
	// The API returns just the ID as a plain string.
	var id string
	if err := json.Unmarshal(data, &id); err != nil {
		// Try as raw string (API returns unquoted UUID).
		id = string(data)
	}
	return id, nil
}
