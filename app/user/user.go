package user

import (
	"github.com/qfpeeeer/gym-buddy-bot/app/exercises"
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

// Manager handles user-related operations
type Manager struct {
	userStorage     Storage
	exerciseStorage ExerciseStorage
}

// NewManager creates a new Manager instance
func NewManager(userStorage Storage, exerciseStorage ExerciseStorage) *Manager {
	return &Manager{
		userStorage:     userStorage,
		exerciseStorage: exerciseStorage,
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
