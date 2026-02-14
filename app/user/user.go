package user

import (
	"context"
	"fmt"

	"github.com/qfpeeeer/gym-buddy-bot/app/analysis"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

// Storage interface defines the methods for user-related storage operations
type Storage interface {
	EnsureUser(telegramID int64) error
	SetHevyAPIKey(telegramID int64, key string) error
	GetHevyAPIKey(telegramID int64) (string, error)
	ClearHevyAPIKey(telegramID int64) error
}

// WorkoutStorage interface defines the methods for workout storage
type WorkoutStorage interface {
	SaveHevyWorkout(userID int64, w hevy.Workout) error
	WorkoutExistsByHevyID(hevyID string) (bool, error)
	GetWorkouts(userID int64, limit int) ([]hevy.Workout, error)
	GetAllWorkouts(userID int64) ([]hevy.Workout, error)
	GetWorkoutsByTitle(userID int64, title string) ([]hevy.Workout, error)
	GetWorkoutCount(userID int64) (int, error)
}

// ExerciseCacheStorage interface defines the methods for exercise template cache
type ExerciseCacheStorage interface {
	UpsertTemplates(templates []hevy.ExerciseTemplate) error
	GetAllTemplates() ([]hevy.ExerciseTemplate, error)
	GetTemplateCount() (int, error)
}

// SyncStorage interface defines the methods for sync state tracking
type SyncStorage interface {
	IsSynced(userID int64) (bool, error)
	UpdateSyncState(userID int64, totalWorkouts int) error
}

// Manager handles user-related operations
type Manager struct {
	userStorage    Storage
	workoutStorage WorkoutStorage
	exerciseCache  ExerciseCacheStorage
	syncStorage    SyncStorage
}

// NewManager creates a new Manager instance
func NewManager(userStorage Storage, workoutStorage WorkoutStorage, exerciseCache ExerciseCacheStorage, syncStorage SyncStorage) *Manager {
	return &Manager{
		userStorage:    userStorage,
		workoutStorage: workoutStorage,
		exerciseCache:  exerciseCache,
		syncStorage:    syncStorage,
	}
}

// EnsureUser ensures a user exists in the storage
func (m *Manager) EnsureUser(telegramID int64) error {
	return m.userStorage.EnsureUser(telegramID)
}

// SetHevyAPIKey stores the Hevy API key for a user
func (m *Manager) SetHevyAPIKey(telegramID int64, key string) error {
	return m.userStorage.SetHevyAPIKey(telegramID, key)
}

// GetHevyAPIKey retrieves the Hevy API key for a user
func (m *Manager) GetHevyAPIKey(telegramID int64) (string, error) {
	return m.userStorage.GetHevyAPIKey(telegramID)
}

// ClearHevyAPIKey removes the Hevy API key for a user
func (m *Manager) ClearHevyAPIKey(telegramID int64) error {
	return m.userStorage.ClearHevyAPIKey(telegramID)
}

// IsHevyConnected checks if a user has a Hevy API key stored
func (m *Manager) IsHevyConnected(telegramID int64) (bool, error) {
	key, err := m.userStorage.GetHevyAPIKey(telegramID)
	if err != nil {
		return false, err
	}
	return key != "", nil
}

// IsSynced checks if a user has completed initial sync
func (m *Manager) IsSynced(userID int64) (bool, error) {
	return m.syncStorage.IsSynced(userID)
}

func (m *Manager) newHevyClient(userID int64) (*hevy.Client, error) {
	key, err := m.userStorage.GetHevyAPIKey(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if key == "" {
		return nil, fmt.Errorf("Hevy account not connected. Use /connect first")
	}
	return hevy.NewClient(key), nil
}

// InitSync performs a full sync: fetches all exercise templates and workouts from Hevy.
func (m *Manager) InitSync(ctx context.Context, userID int64, progress func(string)) error {
	client, err := m.newHevyClient(userID)
	if err != nil {
		return err
	}

	// 1. Fetch and cache exercise templates
	progress("Fetching exercise templates...")
	templates, err := client.GetAllExerciseTemplates(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch exercise templates: %w", err)
	}
	if err := m.exerciseCache.UpsertTemplates(templates); err != nil {
		return fmt.Errorf("failed to cache templates: %w", err)
	}
	progress(fmt.Sprintf("Cached %d exercise templates.", len(templates)))

	// 2. Fetch and save all workouts
	count, err := client.GetWorkoutCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get workout count: %w", err)
	}
	progress(fmt.Sprintf("Importing %d workouts...", count))

	saved := 0
	workouts, err := client.GetAllWorkouts(ctx, func(fetched, totalPages int) {
		progress(fmt.Sprintf("Fetching page %d/%d...", fetched, totalPages))
	})
	if err != nil {
		return fmt.Errorf("failed to fetch workouts: %w", err)
	}

	for _, w := range workouts {
		if err := m.workoutStorage.SaveHevyWorkout(userID, w); err != nil {
			return fmt.Errorf("failed to save workout %s: %w", w.ID, err)
		}
		saved++
		if saved%20 == 0 {
			progress(fmt.Sprintf("Saved %d/%d workouts...", saved, len(workouts)))
		}
	}

	// 3. Update sync state
	localCount, _ := m.workoutStorage.GetWorkoutCount(userID)
	if err := m.syncStorage.UpdateSyncState(userID, localCount); err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	progress(fmt.Sprintf("Done! Synced %d workouts, %d exercise templates cached.\n\nRun /analyze for insights.", saved, len(templates)))
	return nil
}

// IncrementalSync fetches the latest workouts from Hevy and saves new ones.
func (m *Manager) IncrementalSync(ctx context.Context, userID int64) (int, error) {
	client, err := m.newHevyClient(userID)
	if err != nil {
		return 0, err
	}

	resp, err := client.GetWorkouts(ctx, 1, 10)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch workouts: %w", err)
	}

	saved := 0
	for _, w := range resp.Workouts {
		exists, err := m.workoutStorage.WorkoutExistsByHevyID(w.ID)
		if err != nil {
			return saved, err
		}
		if exists {
			continue
		}
		if err := m.workoutStorage.SaveHevyWorkout(userID, w); err != nil {
			return saved, fmt.Errorf("failed to save workout: %w", err)
		}
		saved++
	}

	if saved > 0 {
		localCount, _ := m.workoutStorage.GetWorkoutCount(userID)
		m.syncStorage.UpdateSyncState(userID, localCount)
	}

	return saved, nil
}

// GetLastWorkout fetches the latest workout from Hevy, saves if new, and returns it.
func (m *Manager) GetLastWorkout(ctx context.Context, userID int64) (*hevy.Workout, error) {
	client, err := m.newHevyClient(userID)
	if err != nil {
		return nil, err
	}

	resp, err := client.GetWorkouts(ctx, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest workout: %w", err)
	}
	if len(resp.Workouts) == 0 {
		return nil, nil
	}

	w := resp.Workouts[0]

	// Save if new
	m.workoutStorage.SaveHevyWorkout(userID, w)

	return &w, nil
}

// GetWorkoutsByTitle returns workouts with a given title from local DB.
func (m *Manager) GetWorkoutsByTitle(userID int64, title string) ([]hevy.Workout, error) {
	return m.workoutStorage.GetWorkoutsByTitle(userID, title)
}

// RunAnalysis loads workouts and templates from local DB and runs the analysis engine.
func (m *Manager) RunAnalysis(userID int64, reportType string) (string, error) {
	workouts, err := m.workoutStorage.GetAllWorkouts(userID)
	if err != nil {
		return "", fmt.Errorf("failed to load workouts: %w", err)
	}
	if len(workouts) == 0 {
		return "No workouts found. Run /init first to sync your Hevy data.", nil
	}

	templates, err := m.exerciseCache.GetAllTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to load exercise templates: %w", err)
	}

	result := analysis.Analyze(workouts, templates, 4)

	switch reportType {
	case "volume":
		return analysis.FormatVolumeReport(result.Volume), nil
	case "overload":
		return analysis.FormatOverloadReport(result.Overload), nil
	case "balance":
		return analysis.FormatBalanceReport(result.Balance), nil
	default:
		return analysis.FormatFullReport(result), nil
	}
}
