package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type UserManager interface {
	StoreGoogleSheetsToken(userID int64, token *oauth2.Token) error
	GetGoogleSheetsToken(userID int64) (*oauth2.Token, error)
	SetUserState(userID int64, state string) error
	StoreGoogleSheetID(userID int64, sheetID string) error
}

type GoogleSheetsService struct {
	config      *oauth2.Config
	UserManager UserManager
	TbAPI       *tbapi.BotAPI
	stateStore  map[string]int64 // map[state]userID
}

func NewGoogleSheetsService(clientID, clientSecret, redirectURL string, userManager UserManager, tbAPI *tbapi.BotAPI) *GoogleSheetsService {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{sheets.SpreadsheetsScope},
		Endpoint:     google.Endpoint,
	}

	return &GoogleSheetsService{
		config:      config,
		UserManager: userManager,
		TbAPI:       tbAPI,
		stateStore:  make(map[string]int64),
	}
}

func (s *GoogleSheetsService) GetAuthorizationURL(userID int64) (string, error) {
	// Generate a random state
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Store the state with the user ID
	s.stateStore[state] = userID

	return s.config.AuthCodeURL(state), nil
}

func (s *GoogleSheetsService) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	userID, ok := s.stateStore[state]
	if !ok {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Remove the used state
	delete(s.stateStore, state)

	token, err := s.config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	err = s.UserManager.StoreGoogleSheetsToken(userID, token)
	if err != nil {
		http.Error(w, "Failed to store token", http.StatusInternalServerError)
		return
	}

	// Send a message to the user via the bot
	_, err = s.TbAPI.Send(tbapi.NewMessage(userID, "Google Sheets successfully connected! Now, please share your Google Sheet with this email address: your-service-account@your-project.iam.gserviceaccount.com"))
	if err != nil {
		log.Printf("[error] failed to send message: %v", err)
	}

	// Prompt user to send the Sheet ID
	_, err = s.TbAPI.Send(tbapi.NewMessage(userID, "Great! Now, please send me the ID of your Google Sheet. You can find this in the URL of your sheet: https://docs.google.com/spreadsheets/d/YOUR-SHEET-ID-IS-HERE/edit"))
	if err != nil {
		log.Printf("[error] failed to send message: %v", err)
	}

	// Set user state to waiting for sheet ID
	err = s.UserManager.SetUserState(userID, "waiting_for_sheet_id")
	if err != nil {
		log.Printf("[error] failed to set user state: %v", err)
	}

	// Respond to the user's browser
	_, err = fmt.Fprintf(w, "Authorization successful! You can close this window and return to the Telegram bot.")
	if err != nil {
		log.Printf("[error] failed to write response: %v", err)
	}
}

func (s *GoogleSheetsService) GetSheetsService(userID int64) (*sheets.Service, error) {
	token, err := s.UserManager.GetGoogleSheetsToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	ctx := context.Background()
	return sheets.NewService(ctx, option.WithTokenSource(s.config.TokenSource(ctx, token)))
}

func (s *GoogleSheetsService) WriteToSheet(userID int64, sheetID string, data [][]interface{}) error {
	sheetsService, err := s.GetSheetsService(userID)
	if err != nil {
		return fmt.Errorf("failed to get sheets service: %w", err)
	}

	valueRange := &sheets.ValueRange{
		Values: data,
	}

	_, err = sheetsService.Spreadsheets.Values.Append(sheetID, "Sheet1", valueRange).
		ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("failed to append to sheet: %w", err)
	}

	return nil
}

func (s *GoogleSheetsService) RefreshTokenIfNeeded(userID int64) error {
	token, err := s.UserManager.GetGoogleSheetsToken(userID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	if token.Expiry.Before(time.Now()) {
		newToken, err := s.config.TokenSource(context.Background(), token).Token()
		if err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}

		err = s.UserManager.StoreGoogleSheetsToken(userID, newToken)
		if err != nil {
			return fmt.Errorf("failed to store refreshed token: %w", err)
		}
	}

	return nil
}
