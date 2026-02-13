package hevy

import "time"

// Workout represents a workout from the Hevy API.
type Workout struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	RoutineID   *string           `json:"routine_id"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Exercises   []WorkoutExercise `json:"exercises"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// WorkoutExercise represents an exercise within a workout.
type WorkoutExercise struct {
	Index              int          `json:"index"`
	Title              string       `json:"title"`
	Notes              string       `json:"notes"`
	ExerciseTemplateID string       `json:"exercise_template_id"`
	SupersetID         *int         `json:"superset_id"`
	Sets               []WorkoutSet `json:"sets"`
}

// WorkoutSet represents a single set within an exercise.
type WorkoutSet struct {
	Index           int      `json:"index"`
	Type            string   `json:"type"` // normal, warmup, dropset, failure
	WeightKG        *float64 `json:"weight_kg"`
	Reps            *int     `json:"reps"`
	DistanceMeters  *float64 `json:"distance_meters"`
	DurationSeconds *int     `json:"duration_seconds"`
	RPE             *float64 `json:"rpe"`
	CustomMetric    *float64 `json:"custom_metric"`
}

// ExerciseTemplate represents an exercise template from the Hevy catalog.
type ExerciseTemplate struct {
	ID                    string   `json:"id"`
	Title                 string   `json:"title"`
	Type                  string   `json:"type"` // weight_reps, reps_only, bodyweight_reps, etc.
	PrimaryMuscleGroup    string   `json:"primary_muscle_group"`
	SecondaryMuscleGroups []string `json:"secondary_muscle_groups"`
	Equipment             string   `json:"equipment"`
	IsCustom              bool     `json:"is_custom"`
}

// Routine represents a workout routine/template.
type Routine struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	FolderID  *int              `json:"folder_id"`
	Exercises []RoutineExercise `json:"exercises"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// RoutineExercise represents an exercise within a routine.
type RoutineExercise struct {
	Index              int          `json:"index"`
	Title              string       `json:"title"`
	Notes              *string      `json:"notes"`
	ExerciseTemplateID string       `json:"exercise_template_id"`
	SupersetID         *int         `json:"superset_id"`
	RestSeconds        *int         `json:"rest_seconds"`
	Sets               []RoutineSet `json:"sets"`
}

// RoutineSet represents a set template within a routine exercise.
type RoutineSet struct {
	Index           int      `json:"index"`
	Type            string   `json:"type"`
	WeightKG        *float64 `json:"weight_kg"`
	Reps            *int     `json:"reps"`
	DistanceMeters  *float64 `json:"distance_meters"`
	DurationSeconds *int     `json:"duration_seconds"`
	CustomMetric    *float64 `json:"custom_metric"`
}

// RoutineFolder represents a folder for organizing routines.
type RoutineFolder struct {
	ID        int       `json:"id"`
	Index     int       `json:"index"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// API request types for creating/updating resources.

// CreateWorkoutRequest is the request body for POST /v1/workouts.
type CreateWorkoutRequest struct {
	Workout CreateWorkoutBody `json:"workout"`
}

type CreateWorkoutBody struct {
	Title       string                  `json:"title"`
	Description string                  `json:"description,omitempty"`
	IsPrivate   bool                    `json:"is_private"`
	StartTime   time.Time               `json:"start_time"`
	EndTime     time.Time               `json:"end_time"`
	Exercises   []CreateWorkoutExercise `json:"exercises"`
}

type CreateWorkoutExercise struct {
	ExerciseTemplateID string       `json:"exercise_template_id"`
	Notes              string       `json:"notes,omitempty"`
	SupersetID         *int         `json:"superset_id,omitempty"`
	Sets               []WorkoutSet `json:"sets"`
}

// CreateRoutineRequest is the request body for POST /v1/routines.
type CreateRoutineRequest struct {
	Routine CreateRoutineBody `json:"routine"`
}

type CreateRoutineBody struct {
	Title     string            `json:"title"`
	FolderID  *int              `json:"folder_id,omitempty"`
	Exercises []RoutineExercise `json:"exercises"`
}

// UpdateRoutineRequest is the request body for PUT /v1/routines/{id}.
type UpdateRoutineRequest struct {
	Routine UpdateRoutineBody `json:"routine"`
}

type UpdateRoutineBody struct {
	Title     string            `json:"title"`
	Exercises []RoutineExercise `json:"exercises"`
}

// API response types.

type WorkoutsResponse struct {
	Page      int       `json:"page"`
	PageCount int       `json:"page_count"`
	Workouts  []Workout `json:"workouts"`
}

type WorkoutCountResponse struct {
	WorkoutCount int `json:"workout_count"`
}

type ExerciseTemplatesResponse struct {
	Page              int                `json:"page"`
	PageCount         int                `json:"page_count"`
	ExerciseTemplates []ExerciseTemplate `json:"exercise_templates"`
}

type RoutinesResponse struct {
	Page      int       `json:"page"`
	PageCount int       `json:"page_count"`
	Routines  []Routine `json:"routines"`
}

type RoutineFoldersResponse struct {
	Page      int             `json:"page"`
	PageCount int             `json:"page_count"`
	Routines  []RoutineFolder `json:"routine_folders"`
}

type CreateWorkoutResponse struct {
	Workout []Workout `json:"workout"`
}

type CreateRoutineResponse struct {
	Routine []Routine `json:"routine"`
}

type UpdateRoutineResponse struct {
	Routine []Routine `json:"routine"`
}

type CreateRoutineFolderResponse struct {
	RoutineFolder RoutineFolder `json:"routine_folder"`
}

// CreateExerciseRequest is the request body for POST /v1/exercise_templates.
type CreateExerciseRequest struct {
	Exercise CreateExerciseBody `json:"exercise"`
}

type CreateExerciseBody struct {
	Title             string `json:"title"`
	ExerciseType      string `json:"exercise_type"`
	MuscleGroup       string `json:"muscle_group"`
	EquipmentCategory string `json:"equipment_category"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
