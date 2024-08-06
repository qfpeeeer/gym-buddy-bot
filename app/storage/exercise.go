package storage

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/qfpeeeer/gym-buddy-bot/app/exercises"
	"log"
)

type UserExercise struct {
	ID         int64  `db:"id"`
	UserID     int64  `db:"user_id"`
	ExerciseID string `db:"exercise_id"`
	Name       string `db:"exercise_name"`
	Category   string `db:"exercise_category"`
	CreatedAt  string `db:"created_at"`
}

type ExerciseStorage struct {
	db *sqlx.DB
}

func NewExerciseStorage(db *sqlx.DB) (*ExerciseStorage, error) {
	es := &ExerciseStorage{db: db}
	if err := es.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize exercise storage: %w", err)
	}
	return es, nil
}

func (es *ExerciseStorage) Init() error {
	_, err := es.db.Exec(`
        CREATE TABLE IF NOT EXISTS user_exercises (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            exercise_id TEXT NOT NULL,
            exercise_name TEXT NOT NULL,
            exercise_category TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(telegram_id)
        )
    `)
	if err != nil {
		return fmt.Errorf("failed to create user_exercises table: %w", err)
	}

	_, err = es.db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_user_exercises_user_id ON user_exercises(user_id)
    `)
	if err != nil {
		return fmt.Errorf("failed to create index on user_exercises: %w", err)
	}

	return nil
}

func (es *ExerciseStorage) SetTodayExercises(userID int64, exercises []exercises.Exercise) error {
	tx, err := es.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer a function to handle rollback
	defer func() {
		if p := recover(); p != nil {
			// A panic occurred, attempt to roll back
			tx.Rollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			// Something went wrong, attempt to roll back
			rbErr := tx.Rollback()
			if rbErr != nil {
				// There was an error rolling back, log it
				log.Printf("[error] failed to rollback transaction: %v", rbErr)
			}
		}
	}()

	_, err = tx.Exec("DELETE FROM user_exercises WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete existing exercises: %w", err)
	}

	for _, exercise := range exercises {
		_, err = tx.NamedExec(`
            INSERT INTO user_exercises (user_id, exercise_id, exercise_name, exercise_category) 
            VALUES (:user_id, :exercise_id, :exercise_name, :exercise_category)
        `, map[string]interface{}{
			"user_id":           userID,
			"exercise_id":       exercise.ID,
			"exercise_name":     exercise.Name,
			"exercise_category": exercise.Category,
		})
		if err != nil {
			return fmt.Errorf("failed to insert exercise: %w", err)
		}
	}

	// Attempt to commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
func (es *ExerciseStorage) GetTodayExercises(userID int64) ([]exercises.Exercise, error) {
	var userExercises []UserExercise
	err := es.db.Select(&userExercises, "SELECT * FROM user_exercises WHERE user_id = ? ORDER BY id", userID)
	if err != nil {
		return nil, err
	}

	dbExercises := make([]exercises.Exercise, len(userExercises))
	for i, ue := range userExercises {
		dbExercises[i] = exercises.Exercise{
			ID:       ue.ExerciseID,
			Name:     ue.Name,
			Category: ue.Category,
		}
	}

	return dbExercises, nil
}

func (es *ExerciseStorage) RemoveExercise(userID int64, exerciseIndex int) error {
	_, err := es.db.Exec(`
        DELETE FROM user_exercises 
        WHERE user_id = ? AND id IN (
            SELECT id FROM user_exercises 
            WHERE user_id = ? 
            ORDER BY id 
            LIMIT 1 OFFSET ?
        )
    `, userID, userID, exerciseIndex)
	return err
}

func (es *ExerciseStorage) ReplaceExercise(userID int64, oldExerciseIndex int, newExercise exercises.Exercise) error {
	_, err := es.db.NamedExec(`
        UPDATE user_exercises 
        SET exercise_id = :exercise_id, 
            exercise_name = :exercise_name, 
            exercise_category = :exercise_category 
        WHERE user_id = :user_id AND id IN (
            SELECT id FROM user_exercises 
            WHERE user_id = :user_id 
            ORDER BY id 
            LIMIT 1 OFFSET :offset
        )
    `, map[string]interface{}{
		"exercise_id":       newExercise.ID,
		"exercise_name":     newExercise.Name,
		"exercise_category": newExercise.Category,
		"user_id":           userID,
		"offset":            oldExerciseIndex,
	})
	return err
}
