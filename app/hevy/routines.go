package hevy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetRoutines fetches a page of routines.
func (c *Client) GetRoutines(ctx context.Context, page, pageSize int) (*RoutinesResponse, error) {
	data, err := c.get(ctx, fmt.Sprintf("/routines?page=%d&pageSize=%d", page, pageSize))
	if err != nil {
		return nil, err
	}
	var resp RoutinesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode routines: %w", err)
	}
	return &resp, nil
}

// GetAllRoutines fetches all routines, paginating automatically.
func (c *Client) GetAllRoutines(ctx context.Context) ([]Routine, error) {
	var all []Routine
	page := 1

	for {
		resp, err := c.GetRoutines(ctx, page, defaultPageSize)
		if err != nil {
			return all, fmt.Errorf("hevy: failed to fetch routines page %d: %w", page, err)
		}

		all = append(all, resp.Routines...)

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

// GetRoutineByID fetches a single routine by its ID.
func (c *Client) GetRoutineByID(ctx context.Context, id string) (*Routine, error) {
	data, err := c.get(ctx, "/routines/"+id)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Routine Routine `json:"routine"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode routine: %w", err)
	}
	return &wrapper.Routine, nil
}

// CreateRoutine creates a new routine in Hevy.
func (c *Client) CreateRoutine(ctx context.Context, req CreateRoutineRequest) (*Routine, error) {
	data, err := c.post(ctx, "/routines", req)
	if err != nil {
		return nil, err
	}
	var resp CreateRoutineResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode created routine: %w", err)
	}
	if len(resp.Routine) == 0 {
		return nil, fmt.Errorf("hevy: empty response when creating routine")
	}
	return &resp.Routine[0], nil
}

// UpdateRoutine updates an existing routine in Hevy.
func (c *Client) UpdateRoutine(ctx context.Context, id string, req UpdateRoutineRequest) (*Routine, error) {
	data, err := c.put(ctx, "/routines/"+id, req)
	if err != nil {
		return nil, err
	}
	var resp UpdateRoutineResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode updated routine: %w", err)
	}
	if len(resp.Routine) == 0 {
		return nil, fmt.Errorf("hevy: empty response when updating routine")
	}
	return &resp.Routine[0], nil
}

// GetRoutineFolders fetches a page of routine folders.
func (c *Client) GetRoutineFolders(ctx context.Context, page, pageSize int) (*RoutineFoldersResponse, error) {
	data, err := c.get(ctx, fmt.Sprintf("/routine_folders?page=%d&pageSize=%d", page, pageSize))
	if err != nil {
		return nil, err
	}
	var resp RoutineFoldersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode routine folders: %w", err)
	}
	return &resp, nil
}

// CreateRoutineFolder creates a new routine folder.
func (c *Client) CreateRoutineFolder(ctx context.Context, title string) (*RoutineFolder, error) {
	body := struct {
		RoutineFolder struct {
			Title string `json:"title"`
		} `json:"routine_folder"`
	}{}
	body.RoutineFolder.Title = title

	data, err := c.post(ctx, "/routine_folders", body)
	if err != nil {
		return nil, err
	}
	var resp CreateRoutineFolderResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode created folder: %w", err)
	}
	return &resp.RoutineFolder, nil
}
