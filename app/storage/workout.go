package storage

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/qfpeeeer/gym-buddy-bot/app/strong"
)

type WorkoutStorage struct {
	db *sqlx.DB
}

func NewWorkoutStorage(db *sqlx.DB) (*WorkoutStorage, error) {
	ws := &WorkoutStorage{db: db}
	if err := ws.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize workout storage: %w", err)
	}
	return ws, nil
}

func (ws *WorkoutStorage) Init() error {
	_, err := ws.db.Exec(`
		CREATE TABLE IF NOT EXISTS workouts (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id       INTEGER NOT NULL,
			workout_name  TEXT NOT NULL,
			workout_date  DATETIME NOT NULL,
			duration      TEXT,
			raw_text      TEXT,
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(telegram_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create workouts table: %w", err)
	}

	_, err = ws.db.Exec(`CREATE INDEX IF NOT EXISTS idx_workouts_user_id ON workouts(user_id)`)
	if err != nil {
		return fmt.Errorf("failed to create index on workouts: %w", err)
	}

	_, err = ws.db.Exec(`CREATE INDEX IF NOT EXISTS idx_workouts_user_date ON workouts(user_id, workout_date)`)
	if err != nil {
		return fmt.Errorf("failed to create index on workouts: %w", err)
	}

	_, err = ws.db.Exec(`
		CREATE TABLE IF NOT EXISTS workout_sets (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			workout_id    INTEGER NOT NULL,
			exercise_name TEXT NOT NULL,
			set_order     INTEGER NOT NULL,
			weight        REAL DEFAULT 0,
			reps          INTEGER DEFAULT 0,
			distance      REAL DEFAULT 0,
			seconds       INTEGER DEFAULT 0,
			notes         TEXT,
			FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create workout_sets table: %w", err)
	}

	_, err = ws.db.Exec(`CREATE INDEX IF NOT EXISTS idx_workout_sets_workout_id ON workout_sets(workout_id)`)
	if err != nil {
		return fmt.Errorf("failed to create index on workout_sets: %w", err)
	}

	return nil
}

// SaveWorkout saves a parsed workout for a user. Returns the workout ID.
func (ws *WorkoutStorage) SaveWorkout(userID int64, workout strong.Workout) (int64, error) {
	tx, err := ws.db.Beginx()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			rbErr := tx.Rollback()
			if rbErr != nil {
				log.Printf("[error] failed to rollback transaction: %v", rbErr)
			}
		}
	}()

	result, err := tx.Exec(`
		INSERT INTO workouts (user_id, workout_name, workout_date, duration, raw_text)
		VALUES (?, ?, ?, ?, ?)
	`, userID, workout.Name, workout.Date, workout.Duration, workout.RawText)
	if err != nil {
		return 0, fmt.Errorf("failed to insert workout: %w", err)
	}

	workoutID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get workout id: %w", err)
	}

	for _, exercise := range workout.Exercises {
		for _, set := range exercise.Sets {
			notes := ""
			if set.SetOrder == 1 {
				notes = exercise.Notes
			}
			_, err = tx.Exec(`
				INSERT INTO workout_sets (workout_id, exercise_name, set_order, weight, reps, distance, seconds, notes)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, workoutID, exercise.Name, set.SetOrder, set.Weight, set.Reps, set.Distance, set.Seconds, notes)
			if err != nil {
				return 0, fmt.Errorf("failed to insert workout set: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return workoutID, nil
}

// SaveWorkouts saves multiple workouts, skipping duplicates. Returns the number of new workouts saved.
func (ws *WorkoutStorage) SaveWorkouts(userID int64, workouts []strong.Workout) (int, error) {
	saved := 0
	for _, workout := range workouts {
		isDuplicate, err := ws.CheckDuplicateWorkout(userID, workout.Name, workout.Date)
		if err != nil {
			return saved, fmt.Errorf("failed to check duplicate: %w", err)
		}
		if isDuplicate {
			continue
		}

		if _, err := ws.SaveWorkout(userID, workout); err != nil {
			return saved, fmt.Errorf("failed to save workout %q: %w", workout.Name, err)
		}
		saved++
	}
	return saved, nil
}

// CheckDuplicateWorkout checks if a workout with the same name and date already exists for this user.
func (ws *WorkoutStorage) CheckDuplicateWorkout(userID int64, name string, date time.Time) (bool, error) {
	var count int
	err := ws.db.Get(&count, `
		SELECT COUNT(*) FROM workouts
		WHERE user_id = ? AND workout_name = ? AND workout_date = ?
	`, userID, name, date)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate workout: %w", err)
	}
	return count > 0, nil
}

type workoutRow struct {
	ID          int64  `db:"id"`
	UserID      int64  `db:"user_id"`
	WorkoutName string `db:"workout_name"`
	WorkoutDate string `db:"workout_date"`
	Duration    string `db:"duration"`
	RawText     string `db:"raw_text"`
	CreatedAt   string `db:"created_at"`
}

type workoutSetRow struct {
	ID           int64   `db:"id"`
	WorkoutID    int64   `db:"workout_id"`
	ExerciseName string  `db:"exercise_name"`
	SetOrder     int     `db:"set_order"`
	Weight       float64 `db:"weight"`
	Reps         int     `db:"reps"`
	Distance     float64 `db:"distance"`
	Seconds      int     `db:"seconds"`
	Notes        string  `db:"notes"`
}

// GetRecentWorkouts returns the N most recent workouts for a user.
func (ws *WorkoutStorage) GetRecentWorkouts(userID int64, limit int) ([]strong.Workout, error) {
	var rows []workoutRow
	err := ws.db.Select(&rows, `
		SELECT id, user_id, workout_name, workout_date, COALESCE(duration, '') as duration, COALESCE(raw_text, '') as raw_text, created_at
		FROM workouts
		WHERE user_id = ?
		ORDER BY workout_date DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent workouts: %w", err)
	}

	workouts := make([]strong.Workout, 0, len(rows))
	for _, row := range rows {
		date, _ := time.Parse("2006-01-02 15:04:05", row.WorkoutDate)

		var setRows []workoutSetRow
		err := ws.db.Select(&setRows, `
			SELECT id, workout_id, exercise_name, set_order, weight, reps, distance, seconds, COALESCE(notes, '') as notes
			FROM workout_sets
			WHERE workout_id = ?
			ORDER BY id
		`, row.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get workout sets: %w", err)
		}

		// group sets by exercise name (preserving order)
		var exerciseOrder []string
		exerciseMap := make(map[string]*strong.WorkoutExercise)
		for _, sr := range setRows {
			ex, exists := exerciseMap[sr.ExerciseName]
			if !exists {
				ex = &strong.WorkoutExercise{Name: sr.ExerciseName}
				exerciseMap[sr.ExerciseName] = ex
				exerciseOrder = append(exerciseOrder, sr.ExerciseName)
			}
			if sr.Notes != "" && ex.Notes == "" {
				ex.Notes = sr.Notes
			}
			ex.Sets = append(ex.Sets, strong.ExerciseSet{
				SetOrder: sr.SetOrder,
				Weight:   sr.Weight,
				Reps:     sr.Reps,
				Distance: sr.Distance,
				Seconds:  sr.Seconds,
			})
		}

		exercises := make([]strong.WorkoutExercise, 0, len(exerciseOrder))
		for _, name := range exerciseOrder {
			exercises = append(exercises, *exerciseMap[name])
		}

		workouts = append(workouts, strong.Workout{
			Name:      row.WorkoutName,
			Date:      date,
			Duration:  row.Duration,
			Exercises: exercises,
			RawText:   row.RawText,
		})
	}

	return workouts, nil
}
