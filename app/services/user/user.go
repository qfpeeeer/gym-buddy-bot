package user

import (
	"github.com/qfpeeeer/gym-buddy-bot/app/services/exercises"
	"github.com/qfpeeeer/gym-buddy-bot/app/storage"
	"golang.org/x/oauth2"
)

// Storage interface defines the methods for user-related storage operations
type Storage interface {
	EnsureUser(telegramID int64) error
}

// ExerciseStorage interface defines the methods for exercise-related storage operations
type ExerciseStorage interface {
	SetTodayExercises(telegramID int64, exercises []exercises.Exercise) error
	GetTodayExercises(telegramID int64) ([]exercises.Exercise, error)
	RemoveExercise(telegramID int64, exercise exercises.Exercise) error
	ReplaceExercise(telegramID int64, oldExercise, newExercise exercises.Exercise) error
}

// GoogleSheetsStorage interface defines the methods for google sheets related storage operations
type GoogleSheetsStorage interface {
	StoreGoogleSheetsToken(userID int64, token *oauth2.Token) error
	GetGoogleSheetsToken(userID int64) (*oauth2.Token, error)
	StoreGoogleSheetID(userID int64, sheetID string) error
	GetGoogleSheetID(userID int64) (string, error)
}

// StateStorage interface defines the methods for user state related storage operations
type StateStorage interface {
	SetUserState(userID int64, state string) error
	GetUserState(userID int64) (string, error)
}

// Manager handles user-related operations
type Manager struct {
	userStorage     Storage
	exerciseStorage ExerciseStorage

	googleSheetsStorage GoogleSheetsStorage
	userStateStorage    StateStorage
}

// NewManager creates a new Manager instance
func NewManager(userStorage Storage, exerciseStorage ExerciseStorage, googleSheetsStorage *storage.GoogleSheetsStorage, userStateStorage *storage.UserStateStorage) *Manager {
	return &Manager{
		userStorage:         userStorage,
		exerciseStorage:     exerciseStorage,
		googleSheetsStorage: googleSheetsStorage,
		userStateStorage:    userStateStorage,
	}
}

// EnsureUser ensures a user exists in the storage
func (m *Manager) EnsureUser(telegramID int64) error {
	return m.userStorage.EnsureUser(telegramID)
}

// SetTodayExercises sets the exercises for today for a user
func (m *Manager) SetTodayExercises(telegramID int64, exercises []exercises.Exercise) error {
	return m.exerciseStorage.SetTodayExercises(telegramID, exercises)
}

// GetTodayExercises retrieves today's exercises for a user
func (m *Manager) GetTodayExercises(telegramID int64) ([]exercises.Exercise, error) {
	return m.exerciseStorage.GetTodayExercises(telegramID)
}

// RemoveExercise removes an exercise from a user's today exercises
func (m *Manager) RemoveExercise(telegramID int64, exercise exercises.Exercise) error {
	return m.exerciseStorage.RemoveExercise(telegramID, exercise)
}

// ReplaceExercise replaces an exercise in a user's today exercises
func (m *Manager) ReplaceExercise(telegramID int64, oldExercise, newExercise exercises.Exercise) error {
	return m.exerciseStorage.ReplaceExercise(telegramID, oldExercise, newExercise)
}

func (m *Manager) StoreGoogleSheetsToken(userID int64, token *oauth2.Token) error {
	return m.googleSheetsStorage.StoreGoogleSheetsToken(userID, token)
}

func (m *Manager) GetGoogleSheetsToken(userID int64) (*oauth2.Token, error) {
	return m.googleSheetsStorage.GetGoogleSheetsToken(userID)
}

func (m *Manager) SetUserState(userID int64, state string) error {
	return m.userStateStorage.SetUserState(userID, state)
}

func (m *Manager) GetUserState(userID int64) (string, error) {
	return m.userStateStorage.GetUserState(userID)
}

func (m *Manager) StoreGoogleSheetID(userID int64, sheetID string) error {
	return m.googleSheetsStorage.StoreGoogleSheetID(userID, sheetID)
}

func (m *Manager) GetGoogleSheetID(userID int64) (string, error) {
	return m.googleSheetsStorage.GetGoogleSheetID(userID)
}
