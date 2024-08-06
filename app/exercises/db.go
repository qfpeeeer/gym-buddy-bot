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

// ExerciseDB holds all exercises and provides methods to interact with them
type ExerciseDB struct {
	Exercises map[string]Exercise
}

// NewExerciseDB creates a new ExerciseDB instance
func NewExerciseDB(filePath string) (*ExerciseDB, error) {
	exercises, err := LoadExercises(filePath)
	if err != nil {
		return nil, err
	}

	return &ExerciseDB{Exercises: exercises}, nil
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
func (db *ExerciseDB) GetRandomExercises(count int) []Exercise {
	if count > len(db.Exercises) {
		count = len(db.Exercises)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	exerciseSlice := make([]Exercise, 0, len(db.Exercises))
	for _, exercise := range db.Exercises {
		exerciseSlice = append(exerciseSlice, exercise)
	}

	r.Shuffle(len(exerciseSlice), func(i, j int) {
		exerciseSlice[i], exerciseSlice[j] = exerciseSlice[j], exerciseSlice[i]
	})

	return exerciseSlice[:count]
}

// GetExerciseByName returns an exercise by its name
func (db *ExerciseDB) GetExerciseByName(name string) (Exercise, bool) {
	for _, exercise := range db.Exercises {
		if exercise.Name == name {
			return exercise, true
		}
	}
	return Exercise{}, false
}

// GetExercisesByMuscle returns all exercises that target a specific muscle
func (db *ExerciseDB) GetExercisesByMuscle(muscle string) []Exercise {
	var result []Exercise
	for _, exercise := range db.Exercises {
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
func (db *ExerciseDB) GetExercisesByEquipment(equipment string) []Exercise {
	var result []Exercise
	for _, exercise := range db.Exercises {
		if exercise.Equipment == equipment {
			result = append(result, exercise)
		}
	}
	return result
}

// GetExercisesByLevel returns all exercises of a specific difficulty level
func (db *ExerciseDB) GetExercisesByLevel(level string) []Exercise {
	var result []Exercise
	for _, exercise := range db.Exercises {
		if exercise.Level == level {
			result = append(result, exercise)
		}
	}
	return result
}
