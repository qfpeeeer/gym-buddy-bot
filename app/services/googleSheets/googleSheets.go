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

	// Create and format the new workout sheet
	sheetID, err := s.CreateAndFormatWorkoutSheet(userID)
	if err != nil {
		log.Printf("[error] failed to create and format workout sheet: %v", err)
		http.Error(w, "Failed to create workout sheet", http.StatusInternalServerError)
		return
	}

	// Store the sheet ID for the user
	err = s.UserManager.StoreGoogleSheetID(userID, sheetID)
	if err != nil {
		log.Printf("[error] failed to store sheet ID: %v", err)
		http.Error(w, "Failed to store sheet ID", http.StatusInternalServerError)
		return
	}

	// Send a message to the user via the bot
	_, err = s.TbAPI.Send(tbapi.NewMessage(userID, fmt.Sprintf("Google Sheets successfully connected! A new workout tracker sheet has been created for you. You can access it here: https://docs.google.com/spreadsheets/d/%s", sheetID)))
	if err != nil {
		log.Printf("[error] failed to send message: %v", err)
	}

	// Respond to the user's browser
	_, err = fmt.Fprintf(w, "Authorization successful! A new workout tracker sheet has been created for you. You can close this window and return to the Telegram bot.")
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

func getStringPointer(s string) *string {
	return &s
}

func (s *GoogleSheetsService) CreateAndFormatWorkoutSheet(userID int64) (string, error) {
	sheetsService, err := s.GetSheetsService(userID)
	if err != nil {
		return "", fmt.Errorf("failed to get sheets service: %w", err)
	}

	// Create a new spreadsheet
	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: "My Workout Tracker",
		},
		Sheets: []*sheets.Sheet{
			{
				Properties: &sheets.SheetProperties{
					Title: "Workout Log",
				},
			},
		},
	}

	createdSpreadsheet, err := sheetsService.Spreadsheets.Create(spreadsheet).Do()
	if err != nil {
		return "", fmt.Errorf("failed to create spreadsheet: %w", err)
	}

	sheetID := createdSpreadsheet.SpreadsheetId

	// Get the ID of the first sheet
	firstSheetId := createdSpreadsheet.Sheets[0].Properties.SheetId

	// Format the sheet
	requests := []*sheets.Request{
		// Add headers
		{
			UpdateCells: &sheets.UpdateCellsRequest{
				Rows: []*sheets.RowData{
					{
						Values: []*sheets.CellData{
							{UserEnteredValue: &sheets.ExtendedValue{StringValue: getStringPointer("Date")}},
							{UserEnteredValue: &sheets.ExtendedValue{StringValue: getStringPointer("Exercise")}},
							{UserEnteredValue: &sheets.ExtendedValue{StringValue: getStringPointer("Sets")}},
							{UserEnteredValue: &sheets.ExtendedValue{StringValue: getStringPointer("Reps")}},
							{UserEnteredValue: &sheets.ExtendedValue{StringValue: getStringPointer("Weight")}},
							{UserEnteredValue: &sheets.ExtendedValue{StringValue: getStringPointer("Notes")}},
						},
					},
				},
				Fields: "*",
				Range: &sheets.GridRange{
					SheetId:       firstSheetId,
					StartRowIndex: 0,
					EndRowIndex:   1,
				},
			},
		},
		// Format headers
		{
			RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:       firstSheetId,
					StartRowIndex: 0,
					EndRowIndex:   1,
				},
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{
							Bold: true,
						},
						BackgroundColor: &sheets.Color{
							Red:   0.8,
							Green: 0.8,
							Blue:  0.8,
						},
					},
				},
				Fields: "userEnteredFormat(textFormat,backgroundColor)",
			},
		},
		// Auto-resize columns
		{
			AutoResizeDimensions: &sheets.AutoResizeDimensionsRequest{
				Dimensions: &sheets.DimensionRange{
					SheetId:    firstSheetId,
					Dimension:  "COLUMNS",
					StartIndex: 0,
					EndIndex:   6,
				},
			},
		},
	}

	_, err = sheetsService.Spreadsheets.BatchUpdate(sheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Context(context.Background()).Do()

	if err != nil {
		return "", fmt.Errorf("failed to format spreadsheet: %w", err)
	}

	return sheetID, nil
}
