package storage

import (
	"fmt"
	"github.com/jmoiron/sqlx"
)

type User struct {
	TelegramID int64 `db:"telegram_id"`
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

	return nil
}

func (us *UserStorage) EnsureUser(telegramID int64) error {
	_, err := us.db.Exec("INSERT OR IGNORE INTO users (telegram_id) VALUES (?)", telegramID)
	return err
}

func (us *UserStorage) GetUser(telegramID int64) (*User, error) {
	var user User
	err := us.db.Get(&user, "SELECT telegram_id FROM users WHERE telegram_id = ?", telegramID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
