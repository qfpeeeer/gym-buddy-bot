package user

import (
	"github.com/qfpeeeer/gym-buddy-bot/app/exercises"
	"github.com/qfpeeeer/gym-buddy-bot/app/strong"
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

// WorkoutStorage interface defines the methods for workout-related storage operations
type WorkoutStorage interface {
	SaveWorkout(userID int64, workout strong.Workout) (int64, error)
	SaveWorkouts(userID int64, workouts []strong.Workout) (int, error)
	GetRecentWorkouts(userID int64, limit int) ([]strong.Workout, error)
}

// Manager handles user-related operations
type Manager struct {
	userStorage     Storage
	exerciseStorage ExerciseStorage
	workoutStorage  WorkoutStorage
}

// NewManager creates a new Manager instance
func NewManager(userStorage Storage, exerciseStorage ExerciseStorage, workoutStorage WorkoutStorage) *Manager {
	return &Manager{
		userStorage:     userStorage,
		exerciseStorage: exerciseStorage,
		workoutStorage:  workoutStorage,
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

// SaveWorkout saves a parsed workout for a user
func (m *Manager) SaveWorkout(telegramID int64, workout strong.Workout) (int64, error) {
	return m.workoutStorage.SaveWorkout(telegramID, workout)
}

// SaveWorkouts saves multiple workouts for a user, skipping duplicates
func (m *Manager) SaveWorkouts(telegramID int64, workouts []strong.Workout) (int, error) {
	return m.workoutStorage.SaveWorkouts(telegramID, workouts)
}

// GetRecentWorkouts returns the N most recent workouts for a user
func (m *Manager) GetRecentWorkouts(telegramID int64, limit int) ([]strong.Workout, error) {
	return m.workoutStorage.GetRecentWorkouts(telegramID, limit)
}
