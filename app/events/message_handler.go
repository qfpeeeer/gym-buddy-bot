package events

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
	"github.com/qfpeeeer/gym-buddy-bot/app/strong"
)

type awaitingKeyState struct {
	chatID int64
}

type BotMessageHandler struct {
	TbAPI         TbAPI
	UserManager   UserManager
	TelegramToken string

	mu          sync.Mutex
	awaitingKey map[int64]awaitingKeyState // userID -> state
}

// SetAwaitingHevyKey marks a user as awaiting Hevy API key input.
func (h *BotMessageHandler) SetAwaitingHevyKey(userID, chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.awaitingKey == nil {
		h.awaitingKey = make(map[int64]awaitingKeyState)
	}
	h.awaitingKey[userID] = awaitingKeyState{chatID: chatID}
}

func (h *BotMessageHandler) HandleMessages(ctx context.Context, update tbapi.Update) {
	// handle document uploads (CSV files)
	if update.Message.Document != nil {
		h.handleDocumentUpload(update)
		return
	}

	messageText := update.Message.Text
	if messageText == "" {
		return
	}

	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	log.Printf("[info] received message from user %d", userID)

	// check if user is in "awaiting hevy key" state
	if h.handleHevyKeyInput(ctx, userID, chatID, update.Message.MessageID, messageText) {
		return
	}

	// try to detect if this is a Strong workout text
	if !looksLikeWorkout(messageText) {
		return
	}

	workout, err := strong.ParseText(messageText)
	if err != nil {
		log.Printf("[debug] message from user %d is not a workout: %v", userID, err)
		return
	}

	if err := h.UserManager.EnsureUser(userID); err != nil {
		log.Printf("[error] failed to ensure user: %v", err)
		return
	}

	_, err = h.UserManager.SaveWorkout(userID, *workout)
	if err != nil {
		log.Printf("[error] failed to save workout: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to save workout. Please try again.")
		send(msg, h.TbAPI)
		return
	}

	summary := formatWorkoutSummary(workout)
	msg := tbapi.NewMessage(chatID, summary)
	send(msg, h.TbAPI)
}

func (h *BotMessageHandler) handleDocumentUpload(update tbapi.Update) {
	doc := update.Message.Document
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	if !strings.HasSuffix(strings.ToLower(doc.FileName), ".csv") {
		return
	}

	log.Printf("[info] received CSV upload from user %d: %s", userID, doc.FileName)

	if doc.FileSize > 10*1024*1024 {
		msg := tbapi.NewMessage(chatID, "File is too large. Please upload a CSV under 10MB.")
		send(msg, h.TbAPI)
		return
	}

	file, err := h.TbAPI.GetFile(tbapi.FileConfig{FileID: doc.FileID})
	if err != nil {
		log.Printf("[error] failed to get file: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to download file. Please try again.")
		send(msg, h.TbAPI)
		return
	}

	fileURL := file.Link(h.TelegramToken)
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("[error] failed to download file from telegram: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to download file. Please try again.")
		send(msg, h.TbAPI)
		return
	}
	defer resp.Body.Close()

	csvData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[error] failed to read file content: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to read file. Please try again.")
		send(msg, h.TbAPI)
		return
	}

	workouts, err := strong.ParseCSV(bytes.NewReader(csvData))
	if err != nil {
		log.Printf("[error] failed to parse CSV: %v", err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Failed to parse CSV file: %v", err))
		send(msg, h.TbAPI)
		return
	}

	if len(workouts) == 0 {
		msg := tbapi.NewMessage(chatID, "No workouts found in the CSV file.")
		send(msg, h.TbAPI)
		return
	}

	if err := h.UserManager.EnsureUser(userID); err != nil {
		log.Printf("[error] failed to ensure user: %v", err)
		return
	}

	saved, err := h.UserManager.SaveWorkouts(userID, workouts)
	if err != nil {
		log.Printf("[error] failed to save workouts: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to save workouts. Please try again.")
		send(msg, h.TbAPI)
		return
	}

	skipped := len(workouts) - saved
	text := fmt.Sprintf("Imported %d workout(s) from Strong CSV export.", saved)
	if skipped > 0 {
		text += fmt.Sprintf("\n%d workout(s) skipped (already imported).", skipped)
	}

	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)
}

// handleHevyKeyInput checks if the user is awaiting API key input and processes it.
// Returns true if the message was handled as a key input.
func (h *BotMessageHandler) handleHevyKeyInput(ctx context.Context, userID, chatID int64, messageID int, text string) bool {
	h.mu.Lock()
	state, awaiting := h.awaitingKey[userID]
	if awaiting {
		delete(h.awaitingKey, userID)
	}
	h.mu.Unlock()

	if !awaiting {
		return false
	}

	_ = state // chatID from state could differ, but we use the current chatID

	// Delete the message containing the API key for security.
	deleteMsg := tbapi.NewDeleteMessage(chatID, messageID)
	h.TbAPI.Request(deleteMsg)

	key := strings.TrimSpace(text)
	if key == "" {
		msg := tbapi.NewMessage(chatID, "Empty key. Use /connect to try again.")
		send(msg, h.TbAPI)
		return true
	}

	// Validate the key by calling Hevy API.
	client := hevy.NewClient(key)
	count, err := client.GetWorkoutCount(ctx)
	if err != nil {
		log.Printf("[warn] hevy key validation failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Invalid API key: %v\n\nUse /connect to try again.", err))
		send(msg, h.TbAPI)
		return true
	}

	if err := h.UserManager.EnsureUser(userID); err != nil {
		log.Printf("[error] failed to ensure user: %v", err)
		return true
	}

	if err := h.UserManager.SetHevyAPIKey(userID, key); err != nil {
		log.Printf("[error] failed to save hevy key: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to save API key. Please try again.")
		send(msg, h.TbAPI)
		return true
	}

	msg := tbapi.NewMessage(chatID, fmt.Sprintf("Hevy account connected! You have %d workout(s).\n\nRun /init to import your workout history.", count))
	send(msg, h.TbAPI)
	return true
}

// looksLikeWorkout does a lightweight check to see if the message might be a Strong workout text.
// Checks if line 2 contains a date-like pattern to avoid parsing every random message.
func looksLikeWorkout(text string) bool {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) < 3 {
		return false
	}

	line2 := strings.ToLower(strings.TrimSpace(lines[1]))
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	for _, day := range days {
		if strings.HasPrefix(line2, day) {
			return true
		}
	}

	return false
}

// formatWorkoutSummary creates a human-readable summary of a parsed workout.
func formatWorkoutSummary(workout *strong.Workout) string {
	totalSets := 0
	for _, ex := range workout.Exercises {
		totalSets += len(ex.Sets)
	}

	text := fmt.Sprintf("Workout saved: %s\nDate: %s\n\n%d exercise(s), %d total set(s)\n",
		workout.Name,
		workout.Date.Format("Monday, 2 January 2006"),
		len(workout.Exercises),
		totalSets,
	)

	for _, ex := range workout.Exercises {
		text += fmt.Sprintf("\n  %s - %d set(s)", ex.Name, len(ex.Sets))
	}

	return text
}
