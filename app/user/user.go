package user

import (
	"context"
	"fmt"

	"github.com/qfpeeeer/gym-buddy-bot/app/analysis"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
	"github.com/qfpeeeer/gym-buddy-bot/app/llm"
	"github.com/qfpeeeer/gym-buddy-bot/app/storage"
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

// AISettingsStorage interface defines the methods for AI configuration
type AISettingsStorage interface {
	GetAISettings(userID int64) (*storage.AISettings, error)
	SetOpenAIKey(userID int64, key string) error
	SetModel(userID int64, model string) error
	IsAIConfigured(userID int64) (bool, error)
}

// PreferencesStorage interface defines the methods for user preferences
type PreferencesStorage interface {
	GetPreferences(userID int64) (*storage.UserPreferences, error)
	SetGoals(userID int64, goals string) error
	SetNotes(userID int64, notes string) error
}

// Manager handles user-related operations
type Manager struct {
	userStorage    Storage
	workoutStorage WorkoutStorage
	exerciseCache  ExerciseCacheStorage
	syncStorage    SyncStorage
	aiSettings     AISettingsStorage
	preferences    PreferencesStorage
}

// NewManager creates a new Manager instance
func NewManager(userStorage Storage, workoutStorage WorkoutStorage, exerciseCache ExerciseCacheStorage, syncStorage SyncStorage, aiSettings AISettingsStorage, preferences PreferencesStorage) *Manager {
	return &Manager{
		userStorage:    userStorage,
		workoutStorage: workoutStorage,
		exerciseCache:  exerciseCache,
		syncStorage:    syncStorage,
		aiSettings:     aiSettings,
		preferences:    preferences,
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

// --- AI Settings ---

// SetOpenAIKey stores the OpenAI API key for a user.
func (m *Manager) SetOpenAIKey(userID int64, key string) error {
	return m.aiSettings.SetOpenAIKey(userID, key)
}

// SetAIModel stores the OpenAI model for a user.
func (m *Manager) SetAIModel(userID int64, model string) error {
	return m.aiSettings.SetModel(userID, model)
}

// GetAISettings returns the AI settings for a user.
func (m *Manager) GetAISettings(userID int64) (*storage.AISettings, error) {
	return m.aiSettings.GetAISettings(userID)
}

// IsAIConfigured checks if a user has OpenAI configured.
func (m *Manager) IsAIConfigured(userID int64) (bool, error) {
	return m.aiSettings.IsAIConfigured(userID)
}

// --- User Preferences ---

// SetGoals stores training goals for a user.
func (m *Manager) SetGoals(userID int64, goals string) error {
	return m.preferences.SetGoals(userID, goals)
}

// SetNotes stores training notes for a user.
func (m *Manager) SetNotes(userID int64, notes string) error {
	return m.preferences.SetNotes(userID, notes)
}

// GetPreferences returns the preferences for a user.
func (m *Manager) GetPreferences(userID int64) (*storage.UserPreferences, error) {
	return m.preferences.GetPreferences(userID)
}

// --- AI Advice ---

// GetAdvice generates AI coaching advice based on the user's training data.
func (m *Manager) GetAdvice(ctx context.Context, userID int64) (string, error) {
	return m.askLLM(ctx, userID, "Based on this training data, provide your top 3-5 prioritized recommendations for this client.")
}

// AskQuestion sends a user's question to the LLM with full training context.
func (m *Manager) AskQuestion(ctx context.Context, userID int64, question string) (string, error) {
	return m.askLLM(ctx, userID, question)
}

// GetAnalysisInsights sends analysis results to the LLM for natural language interpretation.
func (m *Manager) GetAnalysisInsights(ctx context.Context, userID int64, analysisText string) (string, error) {
	settings, err := m.aiSettings.GetAISettings(userID)
	if err != nil || settings == nil || settings.OpenAIKey == "" {
		return "", fmt.Errorf("OpenAI not configured. Use /setup_ai first.")
	}

	client := llm.NewClient(settings.OpenAIKey, settings.Model)
	systemPrompt := llm.CoachSystemPrompt + llm.AnalysisInsightsPrompt

	return client.Chat(ctx, systemPrompt, "Here is the mechanical analysis:\n\n"+analysisText+"\n\nExplain the key findings and give actionable recommendations.")
}

func (m *Manager) askLLM(ctx context.Context, userID int64, question string) (string, error) {
	settings, err := m.aiSettings.GetAISettings(userID)
	if err != nil || settings == nil || settings.OpenAIKey == "" {
		return "", fmt.Errorf("OpenAI not configured. Use /setup_ai first.")
	}

	workouts, err := m.workoutStorage.GetAllWorkouts(userID)
	if err != nil {
		return "", fmt.Errorf("failed to load workouts: %w", err)
	}
	if len(workouts) == 0 {
		return "", fmt.Errorf("no workouts found. Run /init first.")
	}

	templates, err := m.exerciseCache.GetAllTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to load templates: %w", err)
	}

	prefs, _ := m.preferences.GetPreferences(userID)
	result := analysis.Analyze(workouts, templates, 4)

	contextStr := llm.BuildContext(workouts, templates, prefs, result)
	userMessage := contextStr + "\n" + question

	client := llm.NewClient(settings.OpenAIKey, settings.Model)
	return client.Chat(ctx, llm.CoachSystemPrompt, userMessage)
}
