package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type SyncState struct {
	UserID        int64     `db:"user_id"`
	LastSync      time.Time `db:"last_sync"`
	TotalWorkouts int       `db:"total_workouts"`
}

type SyncStorage struct {
	db *sqlx.DB
}

func NewSyncStorage(db *sqlx.DB) (*SyncStorage, error) {
	s := &SyncStorage{db: db}
	if err := s.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize sync storage: %w", err)
	}
	return s, nil
}

func (s *SyncStorage) Init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sync_state (
			user_id        INTEGER PRIMARY KEY,
			last_sync      DATETIME,
			total_workouts INTEGER DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(telegram_id)
		)
	`)
	return err
}

// IsSynced returns true if the user has completed initial sync.
func (s *SyncStorage) IsSynced(userID int64) (bool, error) {
	var count int
	err := s.db.Get(&count, `SELECT COUNT(*) FROM sync_state WHERE user_id = ?`, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetSyncState returns the sync state for a user, or nil if not synced yet.
func (s *SyncStorage) GetSyncState(userID int64) (*SyncState, error) {
	var state SyncState
	err := s.db.Get(&state, `SELECT user_id, last_sync, total_workouts FROM sync_state WHERE user_id = ?`, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// UpdateSyncState creates or updates the sync state for a user.
func (s *SyncStorage) UpdateSyncState(userID int64, totalWorkouts int) error {
	_, err := s.db.Exec(`
		INSERT INTO sync_state (user_id, last_sync, total_workouts)
		VALUES (?, CURRENT_TIMESTAMP, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			last_sync = CURRENT_TIMESTAMP,
			total_workouts = ?
	`, userID, totalWorkouts, totalWorkouts)
	return err
}
