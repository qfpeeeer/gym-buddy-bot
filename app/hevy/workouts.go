package hevy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetWorkouts fetches a page of workouts.
func (c *Client) GetWorkouts(ctx context.Context, page, pageSize int) (*WorkoutsResponse, error) {
	data, err := c.get(ctx, fmt.Sprintf("/workouts?page=%d&pageSize=%d", page, pageSize))
	if err != nil {
		return nil, err
	}
	var resp WorkoutsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode workouts: %w", err)
	}
	return &resp, nil
}

// GetAllWorkouts fetches all workouts, paginating automatically.
// The progress callback is called after each page with (fetched so far, total pages).
func (c *Client) GetAllWorkouts(ctx context.Context, progress func(fetched, totalPages int)) ([]Workout, error) {
	var all []Workout
	page := 1

	for {
		resp, err := c.GetWorkouts(ctx, page, defaultPageSize)
		if err != nil {
			return all, fmt.Errorf("hevy: failed to fetch workouts page %d: %w", page, err)
		}

		all = append(all, resp.Workouts...)

		if progress != nil {
			progress(len(all), resp.PageCount)
		}

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

// GetWorkoutByID fetches a single workout by its ID.
func (c *Client) GetWorkoutByID(ctx context.Context, id string) (*Workout, error) {
	data, err := c.get(ctx, "/workouts/"+id)
	if err != nil {
		return nil, err
	}
	var w Workout
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode workout: %w", err)
	}
	return &w, nil
}

// GetWorkoutCount returns the total number of workouts.
func (c *Client) GetWorkoutCount(ctx context.Context) (int, error) {
	data, err := c.get(ctx, "/workouts/count")
	if err != nil {
		return 0, err
	}
	var resp WorkoutCountResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, fmt.Errorf("hevy: failed to decode workout count: %w", err)
	}
	return resp.WorkoutCount, nil
}

// GetWorkoutEvents fetches the workout events feed (for polling changes).
func (c *Client) GetWorkoutEvents(ctx context.Context, page, pageSize int) (*WorkoutsResponse, error) {
	data, err := c.get(ctx, fmt.Sprintf("/workouts/events?page=%d&pageSize=%d", page, pageSize))
	if err != nil {
		return nil, err
	}
	var resp WorkoutsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode workout events: %w", err)
	}
	return &resp, nil
}

// CreateWorkout creates a new workout in Hevy.
func (c *Client) CreateWorkout(ctx context.Context, req CreateWorkoutRequest) (*Workout, error) {
	data, err := c.post(ctx, "/workouts", req)
	if err != nil {
		return nil, err
	}
	var resp CreateWorkoutResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("hevy: failed to decode created workout: %w", err)
	}
	if len(resp.Workout) == 0 {
		return nil, fmt.Errorf("hevy: empty response when creating workout")
	}
	return &resp.Workout[0], nil
}
