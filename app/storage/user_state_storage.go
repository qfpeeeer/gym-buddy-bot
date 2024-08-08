// File: storage/user_state_storage.go

package storage

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type UserState struct {
	UserID int64  `db:"user_id"`
	State  string `db:"state"`
}

type UserStateStorage struct {
	db *sqlx.DB
}

func NewUserStateStorage(db *sqlx.DB) (*UserStateStorage, error) {
	uss := &UserStateStorage{db: db}
	if err := uss.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize user state storage: %w", err)
	}
	return uss, nil
}

func (uss *UserStateStorage) Init() error {
	_, err := uss.db.Exec(`
		CREATE TABLE IF NOT EXISTS user_states (
			user_id INTEGER PRIMARY KEY NOT NULL,
			state TEXT,
			FOREIGN KEY (user_id) REFERENCES users (telegram_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_states table: %w", err)
	}

	return nil
}

func (uss *UserStateStorage) SetUserState(userID int64, state string) error {
	_, err := uss.db.Exec(`
		INSERT INTO user_states (user_id, state)
		VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
		state = excluded.state
	`, userID, state)

	if err != nil {
		return fmt.Errorf("failed to set user state: %w", err)
	}

	return nil
}

func (uss *UserStateStorage) GetUserState(userID int64) (string, error) {
	var state string
	err := uss.db.Get(&state, "SELECT state FROM user_states WHERE user_id = ?", userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user state: %w", err)
	}
	return state, nil
}
