package storage

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type UserPreferences struct {
	UserID int64  `db:"user_id"`
	Goals  string `db:"goals"`
	Notes  string `db:"notes"`
}

type PreferencesStorage struct {
	db *sqlx.DB
}

func NewPreferencesStorage(db *sqlx.DB) (*PreferencesStorage, error) {
	s := &PreferencesStorage{db: db}
	if err := s.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize preferences storage: %w", err)
	}
	return s, nil
}

func (s *PreferencesStorage) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS user_preferences (
			user_id    INTEGER PRIMARY KEY,
			goals      TEXT DEFAULT '',
			notes      TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(telegram_id)
		)
	`)
	return err
}

func (s *PreferencesStorage) GetPreferences(userID int64) (*UserPreferences, error) {
	var prefs UserPreferences
	err := s.db.Get(&prefs, "SELECT user_id, goals, notes FROM user_preferences WHERE user_id = ?", userID)
	if err != nil {
		return &UserPreferences{UserID: userID}, nil
	}
	return &prefs, nil
}

func (s *PreferencesStorage) SetGoals(userID int64, goals string) error {
	_, err := s.db.Exec(`
		INSERT INTO user_preferences (user_id, goals) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET goals = ?, updated_at = CURRENT_TIMESTAMP
	`, userID, goals, goals)
	return err
}

func (s *PreferencesStorage) SetNotes(userID int64, notes string) error {
	_, err := s.db.Exec(`
		INSERT INTO user_preferences (user_id, notes) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET notes = ?, updated_at = CURRENT_TIMESTAMP
	`, userID, notes, notes)
	return err
}
