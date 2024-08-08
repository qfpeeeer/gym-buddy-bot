package exercises

import (
	"encoding/json"
	"math/rand"
	"os"
	"time"
)

// Exercise represents a single exercise with all its details
type Exercise struct {
	Name             string   `json:"name"`
	Force            string   `json:"force"`
	Level            string   `json:"level"`
	Mechanic         string   `json:"mechanic"`
	Equipment        string   `json:"equipment"`
	PrimaryMuscles   []string `json:"primaryMuscles"`
	SecondaryMuscles []string `json:"secondaryMuscles"`
	Instructions     []string `json:"instructions"`
	Category         string   `json:"category"`
	Images           []string `json:"images"`
	ID               string   `json:"id"`
}

// ExerciseManager holds all exercises and provides methods to interact with them
type ExerciseManager struct {
	Exercises map[string]Exercise
}

// NewExerciseManager creates a new ExerciseManager instance
func NewExerciseManager(filePath string) (*ExerciseManager, error) {
	exercises, err := LoadExercises(filePath)
	if err != nil {
		return nil, err
	}

	return &ExerciseManager{Exercises: exercises}, nil
}

// LoadExercises reads the JSON file and returns a map of Exercise
func LoadExercises(filePath string) (map[string]Exercise, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var exercises map[string]Exercise
	if err := json.Unmarshal(data, &exercises); err != nil {
		return nil, err
	}

	return exercises, nil
}

// GetRandomExercises returns a slice of random exercises
func (em *ExerciseManager) GetRandomExercises(count int) []Exercise {
	if count > len(em.Exercises) {
		count = len(em.Exercises)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	exerciseSlice := make([]Exercise, 0, len(em.Exercises))
	for _, exercise := range em.Exercises {
		exerciseSlice = append(exerciseSlice, exercise)
	}

	r.Shuffle(len(exerciseSlice), func(i, j int) {
		exerciseSlice[i], exerciseSlice[j] = exerciseSlice[j], exerciseSlice[i]
	})

	return exerciseSlice[:count]
}

// GetExerciseByName returns an exercise by its name
func (em *ExerciseManager) GetExerciseByName(name string) (Exercise, bool) {
	for _, exercise := range em.Exercises {
		if exercise.Name == name {
			return exercise, true
		}
	}
	return Exercise{}, false
}

// GetExercisesByMuscle returns all exercises that target a specific muscle
func (em *ExerciseManager) GetExercisesByMuscle(muscle string) []Exercise {
	var result []Exercise
	for _, exercise := range em.Exercises {
		for _, primaryMuscle := range exercise.PrimaryMuscles {
			if primaryMuscle == muscle {
				result = append(result, exercise)
				break
			}
		}
	}
	return result
}

// GetExercisesByEquipment returns all exercises that use specific equipment
func (em *ExerciseManager) GetExercisesByEquipment(equipment string) []Exercise {
	var result []Exercise
	for _, exercise := range em.Exercises {
		if exercise.Equipment == equipment {
			result = append(result, exercise)
		}
	}
	return result
}

// GetExercisesByLevel returns all exercises of a specific difficulty level
func (em *ExerciseManager) GetExercisesByLevel(level string) []Exercise {
	var result []Exercise
	for _, exercise := range em.Exercises {
		if exercise.Level == level {
			result = append(result, exercise)
		}
	}
	return result
}

// GetExerciseByID returns an exercise by its ID
func (em *ExerciseManager) GetExerciseByID(id string) (Exercise, bool) {
	exercise, found := em.Exercises[id]
	return exercise, found
}
