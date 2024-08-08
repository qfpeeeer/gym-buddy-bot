// File: storage/google_sheets_storage.go

package storage

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
)

type GoogleSheetsData struct {
	UserID       int64     `db:"user_id"`
	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	TokenType    string    `db:"token_type"`
	Expiry       time.Time `db:"expiry"`
	SheetID      string    `db:"sheet_id"`
}

type GoogleSheetsStorage struct {
	db *sqlx.DB
}

func NewGoogleSheetsStorage(db *sqlx.DB) (*GoogleSheetsStorage, error) {
	gss := &GoogleSheetsStorage{db: db}
	if err := gss.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize Google Sheets storage: %w", err)
	}
	return gss, nil
}

func (gss *GoogleSheetsStorage) Init() error {
	_, err := gss.db.Exec(`
		CREATE TABLE IF NOT EXISTS user_google_sheets (
			user_id INTEGER PRIMARY KEY NOT NULL,
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			token_type TEXT NOT NULL,
			expiry DATETIME NOT NULL,
			sheet_id TEXT,
			FOREIGN KEY (user_id) REFERENCES users (telegram_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_google_sheets table: %w", err)
	}

	return nil
}

func (gss *GoogleSheetsStorage) StoreGoogleSheetsToken(userID int64, token *oauth2.Token) error {
	_, err := gss.db.NamedExec(`
		INSERT INTO user_google_sheets (user_id, access_token, refresh_token, token_type, expiry)
		VALUES (:user_id, :access_token, :refresh_token, :token_type, :expiry)
		ON CONFLICT(user_id) DO UPDATE SET
		access_token = :access_token,
		refresh_token = :refresh_token,
		token_type = :token_type,
		expiry = :expiry
	`, map[string]interface{}{
		"user_id":       userID,
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
		"expiry":        token.Expiry,
	})

	if err != nil {
		return fmt.Errorf("failed to store Google Sheets token: %w", err)
	}

	return nil
}

func (gss *GoogleSheetsStorage) GetGoogleSheetsToken(userID int64) (*oauth2.Token, error) {
	var data GoogleSheetsData
	err := gss.db.Get(&data, "SELECT user_id, access_token, refresh_token, token_type, expiry FROM user_google_sheets WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Google Sheets token: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		TokenType:    data.TokenType,
		Expiry:       data.Expiry,
	}, nil
}

func (gss *GoogleSheetsStorage) StoreGoogleSheetID(userID int64, sheetID string) error {
	_, err := gss.db.Exec("UPDATE user_google_sheets SET sheet_id = ? WHERE user_id = ?", sheetID, userID)
	if err != nil {
		return fmt.Errorf("failed to store Google Sheet ID: %w", err)
	}
	return nil
}

func (gss *GoogleSheetsStorage) GetGoogleSheetID(userID int64) (string, error) {
	var sheetID string
	err := gss.db.Get(&sheetID, "SELECT sheet_id FROM user_google_sheets WHERE user_id = ?", userID)
	if err != nil {
		return "", fmt.Errorf("failed to get Google Sheet ID: %w", err)
	}
	return sheetID, nil
}
