package storage

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type User struct {
	TelegramID int64   `db:"telegram_id"`
	HevyAPIKey *string `db:"hevy_api_key"`
}

type UserStorage struct {
	db *sqlx.DB
}

func NewUserStorage(db *sqlx.DB) (*UserStorage, error) {
	us := &UserStorage{db: db}
	if err := us.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize user storage: %w", err)
	}
	return us, nil
}

func (us *UserStorage) Init() error {
	_, err := us.db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            telegram_id INTEGER PRIMARY KEY NOT NULL
        )
    `)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Add hevy_api_key column if it doesn't exist (migration).
	us.db.Exec(`ALTER TABLE users ADD COLUMN hevy_api_key TEXT`)

	return nil
}

func (us *UserStorage) EnsureUser(telegramID int64) error {
	_, err := us.db.Exec("INSERT OR IGNORE INTO users (telegram_id) VALUES (?)", telegramID)
	return err
}

func (us *UserStorage) GetUser(telegramID int64) (*User, error) {
	var user User
	err := us.db.Get(&user, "SELECT telegram_id, hevy_api_key FROM users WHERE telegram_id = ?", telegramID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (us *UserStorage) SetHevyAPIKey(telegramID int64, key string) error {
	_, err := us.db.Exec("UPDATE users SET hevy_api_key = ? WHERE telegram_id = ?", key, telegramID)
	return err
}

func (us *UserStorage) GetHevyAPIKey(telegramID int64) (string, error) {
	var key *string
	err := us.db.Get(&key, "SELECT hevy_api_key FROM users WHERE telegram_id = ?", telegramID)
	if err != nil {
		return "", err
	}
	if key == nil {
		return "", nil
	}
	return *key, nil
}

func (us *UserStorage) ClearHevyAPIKey(telegramID int64) error {
	_, err := us.db.Exec("UPDATE users SET hevy_api_key = NULL WHERE telegram_id = ?", telegramID)
	return err
}
