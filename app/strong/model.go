package strong

import "time"

// Workout represents a complete workout session from Strong.
type Workout struct {
	Name      string
	Date      time.Time
	Duration  string
	Exercises []WorkoutExercise
	RawText   string
}

// WorkoutExercise represents one exercise within a workout, with all its sets.
type WorkoutExercise struct {
	Name  string
	Notes string
	Sets  []ExerciseSet
}

// ExerciseSet represents a single set of an exercise.
type ExerciseSet struct {
	SetOrder int
	Weight   float64
	Reps     int
	Distance float64
	Seconds  int
}
