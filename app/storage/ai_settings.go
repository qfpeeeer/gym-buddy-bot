package storage

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type AISettings struct {
	UserID    int64  `db:"user_id"`
	OpenAIKey string `db:"openai_key"`
	Model     string `db:"model"`
}

type AISettingsStorage struct {
	db *sqlx.DB
}

func NewAISettingsStorage(db *sqlx.DB) (*AISettingsStorage, error) {
	s := &AISettingsStorage{db: db}
	if err := s.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize ai_settings storage: %w", err)
	}
	return s, nil
}

func (s *AISettingsStorage) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_settings (
			user_id    INTEGER PRIMARY KEY,
			openai_key TEXT,
			model      TEXT DEFAULT 'gpt-4.1-mini',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(telegram_id)
		)
	`)
	return err
}

func (s *AISettingsStorage) GetAISettings(userID int64) (*AISettings, error) {
	var settings AISettings
	err := s.db.Get(&settings, "SELECT user_id, openai_key, model FROM ai_settings WHERE user_id = ?", userID)
	if err != nil {
		return nil, nil
	}
	return &settings, nil
}

func (s *AISettingsStorage) SetOpenAIKey(userID int64, key string) error {
	_, err := s.db.Exec(`
		INSERT INTO ai_settings (user_id, openai_key) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET openai_key = ?, updated_at = CURRENT_TIMESTAMP
	`, userID, key, key)
	return err
}

func (s *AISettingsStorage) SetModel(userID int64, model string) error {
	_, err := s.db.Exec(`
		INSERT INTO ai_settings (user_id, model) VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET model = ?, updated_at = CURRENT_TIMESTAMP
	`, userID, model, model)
	return err
}

func (s *AISettingsStorage) IsAIConfigured(userID int64) (bool, error) {
	settings, err := s.GetAISettings(userID)
	if err != nil {
		return false, err
	}
	return settings != nil && settings.OpenAIKey != "", nil
}
