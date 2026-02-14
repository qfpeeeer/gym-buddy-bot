package storage

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

type ExerciseCacheStorage struct {
	db *sqlx.DB
}

func NewExerciseCacheStorage(db *sqlx.DB) (*ExerciseCacheStorage, error) {
	s := &ExerciseCacheStorage{db: db}
	if err := s.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize exercise cache: %w", err)
	}
	return s, nil
}

func (s *ExerciseCacheStorage) Init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS exercise_templates (
			id                      TEXT PRIMARY KEY,
			title                   TEXT NOT NULL,
			type                    TEXT NOT NULL,
			primary_muscle_group    TEXT,
			secondary_muscle_groups TEXT,
			equipment               TEXT,
			is_custom               INTEGER DEFAULT 0,
			fetched_at              DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// UpsertTemplates bulk-inserts or updates exercise templates.
func (s *ExerciseCacheStorage) UpsertTemplates(templates []hevy.ExerciseTemplate) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO exercise_templates
		(id, title, type, primary_muscle_group, secondary_muscle_groups, equipment, is_custom, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, t := range templates {
		secondary, _ := json.Marshal(t.SecondaryMuscleGroups)
		isCustom := 0
		if t.IsCustom {
			isCustom = 1
		}
		_, err = stmt.Exec(t.ID, t.Title, t.Type, t.PrimaryMuscleGroup, string(secondary), t.Equipment, isCustom)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert template %s: %w", t.ID, err)
		}
	}

	return tx.Commit()
}

type templateRow struct {
	ID                    string  `db:"id"`
	Title                 string  `db:"title"`
	Type                  string  `db:"type"`
	PrimaryMuscleGroup    *string `db:"primary_muscle_group"`
	SecondaryMuscleGroups *string `db:"secondary_muscle_groups"`
	Equipment             *string `db:"equipment"`
	IsCustom              int     `db:"is_custom"`
}

func (r *templateRow) toTemplate() hevy.ExerciseTemplate {
	t := hevy.ExerciseTemplate{
		ID:       r.ID,
		Title:    r.Title,
		Type:     r.Type,
		IsCustom: r.IsCustom == 1,
	}
	if r.PrimaryMuscleGroup != nil {
		t.PrimaryMuscleGroup = *r.PrimaryMuscleGroup
	}
	if r.Equipment != nil {
		t.Equipment = *r.Equipment
	}
	if r.SecondaryMuscleGroups != nil {
		json.Unmarshal([]byte(*r.SecondaryMuscleGroups), &t.SecondaryMuscleGroups)
	}
	if t.SecondaryMuscleGroups == nil {
		t.SecondaryMuscleGroups = []string{}
	}
	return t
}

// GetTemplate returns a single exercise template by ID.
func (s *ExerciseCacheStorage) GetTemplate(id string) (*hevy.ExerciseTemplate, error) {
	var row templateRow
	err := s.db.Get(&row, `SELECT id, title, type, primary_muscle_group, secondary_muscle_groups, equipment, is_custom FROM exercise_templates WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	t := row.toTemplate()
	return &t, nil
}

// GetAllTemplates returns all cached exercise templates.
func (s *ExerciseCacheStorage) GetAllTemplates() ([]hevy.ExerciseTemplate, error) {
	var rows []templateRow
	err := s.db.Select(&rows, `SELECT id, title, type, primary_muscle_group, secondary_muscle_groups, equipment, is_custom FROM exercise_templates ORDER BY title`)
	if err != nil {
		return nil, err
	}
	templates := make([]hevy.ExerciseTemplate, len(rows))
	for i, r := range rows {
		templates[i] = r.toTemplate()
	}
	return templates, nil
}

// GetTemplateCount returns the number of cached templates.
func (s *ExerciseCacheStorage) GetTemplateCount() (int, error) {
	var count int
	err := s.db.Get(&count, `SELECT COUNT(*) FROM exercise_templates`)
	return count, err
}
