package storage

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
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
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL,
			hevy_id    TEXT NOT NULL,
			title      TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time   DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(telegram_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create workouts table: %w", err)
	}

	_, err = ws.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_workouts_hevy_id ON workouts(hevy_id)`)
	if err != nil {
		return fmt.Errorf("failed to create hevy_id index: %w", err)
	}

	_, err = ws.db.Exec(`CREATE INDEX IF NOT EXISTS idx_workouts_user_id ON workouts(user_id)`)
	if err != nil {
		return fmt.Errorf("failed to create user_id index: %w", err)
	}

	_, err = ws.db.Exec(`
		CREATE TABLE IF NOT EXISTS workout_exercises (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			workout_id           INTEGER NOT NULL,
			exercise_index       INTEGER NOT NULL,
			title                TEXT NOT NULL,
			exercise_template_id TEXT NOT NULL,
			notes                TEXT,
			superset_id          INTEGER,
			FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create workout_exercises table: %w", err)
	}

	_, err = ws.db.Exec(`
		CREATE TABLE IF NOT EXISTS workout_sets (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			exercise_id INTEGER NOT NULL,
			set_index   INTEGER NOT NULL,
			set_type    TEXT DEFAULT 'normal',
			weight_kg   REAL,
			reps        INTEGER,
			distance_m  REAL,
			duration_s  INTEGER,
			rpe         REAL,
			FOREIGN KEY (exercise_id) REFERENCES workout_exercises(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create workout_sets table: %w", err)
	}

	return nil
}

// SaveHevyWorkout saves a workout from Hevy API to local DB. Skips if hevy_id already exists.
func (ws *WorkoutStorage) SaveHevyWorkout(userID int64, w hevy.Workout) error {
	exists, err := ws.WorkoutExistsByHevyID(w.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	tx, err := ws.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("[error] failed to rollback: %v", rbErr)
			}
		}
	}()

	result, err := tx.Exec(`
		INSERT INTO workouts (user_id, hevy_id, title, start_time, end_time)
		VALUES (?, ?, ?, ?, ?)
	`, userID, w.ID, w.Title, w.StartTime, w.EndTime)
	if err != nil {
		return fmt.Errorf("failed to insert workout: %w", err)
	}

	workoutID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get workout id: %w", err)
	}

	for _, ex := range w.Exercises {
		exResult, err := tx.Exec(`
			INSERT INTO workout_exercises (workout_id, exercise_index, title, exercise_template_id, notes, superset_id)
			VALUES (?, ?, ?, ?, ?, ?)
		`, workoutID, ex.Index, ex.Title, ex.ExerciseTemplateID, ex.Notes, ex.SupersetID)
		if err != nil {
			return fmt.Errorf("failed to insert exercise: %w", err)
		}

		exerciseID, err := exResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get exercise id: %w", err)
		}

		for _, s := range ex.Sets {
			_, err = tx.Exec(`
				INSERT INTO workout_sets (exercise_id, set_index, set_type, weight_kg, reps, distance_m, duration_s, rpe)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, exerciseID, s.Index, s.Type, s.WeightKG, s.Reps, s.DistanceMeters, s.DurationSeconds, s.RPE)
			if err != nil {
				return fmt.Errorf("failed to insert set: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// WorkoutExistsByHevyID checks if a workout with the given Hevy ID already exists.
func (ws *WorkoutStorage) WorkoutExistsByHevyID(hevyID string) (bool, error) {
	var count int
	err := ws.db.Get(&count, `SELECT COUNT(*) FROM workouts WHERE hevy_id = ?`, hevyID)
	return count > 0, err
}

// GetWorkoutCount returns the total number of workouts for a user.
func (ws *WorkoutStorage) GetWorkoutCount(userID int64) (int, error) {
	var count int
	err := ws.db.Get(&count, `SELECT COUNT(*) FROM workouts WHERE user_id = ?`, userID)
	return count, err
}

type workoutRow struct {
	ID        int64  `db:"id"`
	HevyID    string `db:"hevy_id"`
	Title     string `db:"title"`
	StartTime string `db:"start_time"`
	EndTime   string `db:"end_time"`
}

type exerciseRow struct {
	ID                 int64   `db:"id"`
	WorkoutID          int64   `db:"workout_id"`
	ExerciseIndex      int     `db:"exercise_index"`
	Title              string  `db:"title"`
	ExerciseTemplateID string  `db:"exercise_template_id"`
	Notes              *string `db:"notes"`
	SupersetID         *int    `db:"superset_id"`
}

type setRow struct {
	ExerciseID int      `db:"exercise_id"`
	SetIndex   int      `db:"set_index"`
	SetType    string   `db:"set_type"`
	WeightKG   *float64 `db:"weight_kg"`
	Reps       *int     `db:"reps"`
	DistanceM  *float64 `db:"distance_m"`
	DurationS  *int     `db:"duration_s"`
	RPE        *float64 `db:"rpe"`
}

// GetWorkouts returns the N most recent workouts for a user with full exercise/set data.
func (ws *WorkoutStorage) GetWorkouts(userID int64, limit int) ([]hevy.Workout, error) {
	var rows []workoutRow
	err := ws.db.Select(&rows, `
		SELECT id, hevy_id, title, start_time, end_time
		FROM workouts WHERE user_id = ?
		ORDER BY start_time DESC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get workouts: %w", err)
	}
	return ws.hydrateWorkouts(rows)
}

// GetAllWorkouts returns all workouts for a user ordered by start_time ASC.
func (ws *WorkoutStorage) GetAllWorkouts(userID int64) ([]hevy.Workout, error) {
	var rows []workoutRow
	err := ws.db.Select(&rows, `
		SELECT id, hevy_id, title, start_time, end_time
		FROM workouts WHERE user_id = ?
		ORDER BY start_time ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workouts: %w", err)
	}
	return ws.hydrateWorkouts(rows)
}

// GetWorkoutsByTitle returns all workouts with a given title for a user.
func (ws *WorkoutStorage) GetWorkoutsByTitle(userID int64, title string) ([]hevy.Workout, error) {
	var rows []workoutRow
	err := ws.db.Select(&rows, `
		SELECT id, hevy_id, title, start_time, end_time
		FROM workouts WHERE user_id = ? AND title = ?
		ORDER BY start_time DESC
	`, userID, title)
	if err != nil {
		return nil, fmt.Errorf("failed to get workouts by title: %w", err)
	}
	return ws.hydrateWorkouts(rows)
}

func (ws *WorkoutStorage) hydrateWorkouts(rows []workoutRow) ([]hevy.Workout, error) {
	workouts := make([]hevy.Workout, 0, len(rows))
	for _, r := range rows {
		startTime, _ := time.Parse("2006-01-02T15:04:05Z", r.StartTime)
		if startTime.IsZero() {
			startTime, _ = time.Parse("2006-01-02 15:04:05", r.StartTime)
		}
		if startTime.IsZero() {
			startTime, _ = time.Parse(time.RFC3339, r.StartTime)
		}
		endTime, _ := time.Parse("2006-01-02T15:04:05Z", r.EndTime)
		if endTime.IsZero() {
			endTime, _ = time.Parse("2006-01-02 15:04:05", r.EndTime)
		}
		if endTime.IsZero() {
			endTime, _ = time.Parse(time.RFC3339, r.EndTime)
		}

		var exRows []exerciseRow
		err := ws.db.Select(&exRows, `
			SELECT id, workout_id, exercise_index, title, exercise_template_id, notes, superset_id
			FROM workout_exercises WHERE workout_id = ?
			ORDER BY exercise_index
		`, r.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get exercises: %w", err)
		}

		exercises := make([]hevy.WorkoutExercise, 0, len(exRows))
		for _, er := range exRows {
			var sRows []setRow
			err := ws.db.Select(&sRows, `
				SELECT exercise_id, set_index, COALESCE(set_type, 'normal') as set_type,
				       weight_kg, reps, distance_m, duration_s, rpe
				FROM workout_sets WHERE exercise_id = ?
				ORDER BY set_index
			`, er.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get sets: %w", err)
			}

			sets := make([]hevy.WorkoutSet, 0, len(sRows))
			for _, sr := range sRows {
				sets = append(sets, hevy.WorkoutSet{
					Index:           sr.SetIndex,
					Type:            sr.SetType,
					WeightKG:        sr.WeightKG,
					Reps:            sr.Reps,
					DistanceMeters:  sr.DistanceM,
					DurationSeconds: sr.DurationS,
					RPE:             sr.RPE,
				})
			}

			notes := ""
			if er.Notes != nil {
				notes = *er.Notes
			}

			exercises = append(exercises, hevy.WorkoutExercise{
				Index:              er.ExerciseIndex,
				Title:              er.Title,
				ExerciseTemplateID: er.ExerciseTemplateID,
				Notes:              notes,
				SupersetID:         er.SupersetID,
				Sets:               sets,
			})
		}

		workouts = append(workouts, hevy.Workout{
			ID:        r.HevyID,
			Title:     r.Title,
			StartTime: startTime,
			EndTime:   endTime,
			Exercises: exercises,
		})
	}
	return workouts, nil
}
