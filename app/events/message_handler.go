package events

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

type awaitingState struct {
	chatID int64
}

type BotMessageHandler struct {
	TbAPI       TbAPI
	UserManager UserManager

	// GenerateAndPreview is set by main.go to call through to the callback handler's logic.
	GenerateAndPreview func(ctx context.Context, userID, chatID int64, tplType, typeName, extraNotes string)

	mu                 sync.Mutex
	awaitingHevyKey    map[int64]awaitingState
	awaitingAIKey      map[int64]awaitingState
	awaitingNotes      map[int64]awaitingState
	awaitingGoalText   map[int64]awaitingState
	awaitingCustomTpl  map[int64]awaitingState
	awaitingRegenNotes map[int64]awaitingState
}

// SetAwaitingHevyKey marks a user as awaiting Hevy API key input.
func (h *BotMessageHandler) SetAwaitingHevyKey(userID, chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.awaitingHevyKey == nil {
		h.awaitingHevyKey = make(map[int64]awaitingState)
	}
	h.awaitingHevyKey[userID] = awaitingState{chatID: chatID}
}

// SetAwaitingOpenAIKey marks a user as awaiting OpenAI API key input.
func (h *BotMessageHandler) SetAwaitingOpenAIKey(userID, chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.awaitingAIKey == nil {
		h.awaitingAIKey = make(map[int64]awaitingState)
	}
	h.awaitingAIKey[userID] = awaitingState{chatID: chatID}
}

// SetAwaitingNotes marks a user as awaiting training notes input.
func (h *BotMessageHandler) SetAwaitingNotes(userID, chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.awaitingNotes == nil {
		h.awaitingNotes = make(map[int64]awaitingState)
	}
	h.awaitingNotes[userID] = awaitingState{chatID: chatID}
}

// SetAwaitingGoalText marks a user as awaiting custom goal text input.
func (h *BotMessageHandler) SetAwaitingGoalText(userID, chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.awaitingGoalText == nil {
		h.awaitingGoalText = make(map[int64]awaitingState)
	}
	h.awaitingGoalText[userID] = awaitingState{chatID: chatID}
}

// SetAwaitingCustomTplType marks a user as awaiting custom template type text.
func (h *BotMessageHandler) SetAwaitingCustomTplType(userID, chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.awaitingCustomTpl == nil {
		h.awaitingCustomTpl = make(map[int64]awaitingState)
	}
	h.awaitingCustomTpl[userID] = awaitingState{chatID: chatID}
}

// SetAwaitingRegenNotes marks a user as awaiting regen notes text.
func (h *BotMessageHandler) SetAwaitingRegenNotes(userID, chatID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.awaitingRegenNotes == nil {
		h.awaitingRegenNotes = make(map[int64]awaitingState)
	}
	h.awaitingRegenNotes[userID] = awaitingState{chatID: chatID}
}

func (h *BotMessageHandler) HandleMessages(ctx context.Context, update tbapi.Update) {
	messageText := update.Message.Text
	if messageText == "" {
		return
	}

	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	log.Printf("[info] received message from user %d", userID)

	if h.handleHevyKeyInput(ctx, userID, chatID, update.Message.MessageID, messageText) {
		return
	}
	if h.handleOpenAIKeyInput(userID, chatID, update.Message.MessageID, messageText) {
		return
	}
	if h.handleNotesInput(userID, chatID, messageText) {
		return
	}
	if h.handleGoalTextInput(userID, chatID, messageText) {
		return
	}
	if h.handleCustomTplInput(ctx, userID, chatID, messageText) {
		return
	}
	if h.handleRegenNotesInput(ctx, userID, chatID, messageText) {
		return
	}
}

// handleHevyKeyInput checks if the user is awaiting API key input and processes it.
// Returns true if the message was handled as a key input.
func (h *BotMessageHandler) handleHevyKeyInput(ctx context.Context, userID, chatID int64, messageID int, text string) bool {
	h.mu.Lock()
	_, awaiting := h.awaitingHevyKey[userID]
	if awaiting {
		delete(h.awaitingHevyKey, userID)
	}
	h.mu.Unlock()

	if !awaiting {
		return false
	}

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

func (h *BotMessageHandler) handleOpenAIKeyInput(userID, chatID int64, messageID int, text string) bool {
	h.mu.Lock()
	_, awaiting := h.awaitingAIKey[userID]
	if awaiting {
		delete(h.awaitingAIKey, userID)
	}
	h.mu.Unlock()

	if !awaiting {
		return false
	}

	// Delete the message containing the API key for security.
	deleteMsg := tbapi.NewDeleteMessage(chatID, messageID)
	h.TbAPI.Request(deleteMsg)

	key := strings.TrimSpace(text)
	if key == "" {
		msg := tbapi.NewMessage(chatID, "Empty key. Use /setup_ai to try again.")
		send(msg, h.TbAPI)
		return true
	}

	if err := h.UserManager.SetOpenAIKey(userID, key); err != nil {
		log.Printf("[error] failed to save OpenAI key: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to save API key. Please try again.")
		send(msg, h.TbAPI)
		return true
	}

	// Show model picker
	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("gpt-4.1-nano (fastest)", "set_model_gpt-4.1-nano"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("gpt-4.1-mini (recommended)", "set_model_gpt-4.1-mini"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("gpt-4.1 (most capable)", "set_model_gpt-4.1"),
		),
	)

	msg := tbapi.NewMessage(chatID, "OpenAI key saved! Choose a model:")
	msg.ReplyMarkup = keyboard
	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send model picker: %v", err)
	}
	return true
}

func (h *BotMessageHandler) handleNotesInput(userID, chatID int64, text string) bool {
	h.mu.Lock()
	_, awaiting := h.awaitingNotes[userID]
	if awaiting {
		delete(h.awaitingNotes, userID)
	}
	h.mu.Unlock()

	if !awaiting {
		return false
	}

	notes := strings.TrimSpace(text)
	if notes == "" {
		msg := tbapi.NewMessage(chatID, "Empty notes. Use /notes to try again.")
		send(msg, h.TbAPI)
		return true
	}

	if err := h.UserManager.SetNotes(userID, notes); err != nil {
		log.Printf("[error] failed to save notes: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to save notes.")
		send(msg, h.TbAPI)
		return true
	}

	msg := tbapi.NewMessage(chatID, fmt.Sprintf("Notes saved: %s", notes))
	send(msg, h.TbAPI)
	return true
}

func (h *BotMessageHandler) handleGoalTextInput(userID, chatID int64, text string) bool {
	h.mu.Lock()
	_, awaiting := h.awaitingGoalText[userID]
	if awaiting {
		delete(h.awaitingGoalText, userID)
	}
	h.mu.Unlock()

	if !awaiting {
		return false
	}

	goal := strings.TrimSpace(text)
	if goal == "" {
		msg := tbapi.NewMessage(chatID, "Empty goal. Use /goals to try again.")
		send(msg, h.TbAPI)
		return true
	}

	if err := h.UserManager.SetGoals(userID, goal); err != nil {
		log.Printf("[error] failed to save goal: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to save goal.")
		send(msg, h.TbAPI)
		return true
	}

	msg := tbapi.NewMessage(chatID, fmt.Sprintf("Goal set: %s", goal))
	send(msg, h.TbAPI)
	return true
}

func (h *BotMessageHandler) handleCustomTplInput(ctx context.Context, userID, chatID int64, text string) bool {
	h.mu.Lock()
	_, awaiting := h.awaitingCustomTpl[userID]
	if awaiting {
		delete(h.awaitingCustomTpl, userID)
	}
	h.mu.Unlock()

	if !awaiting {
		return false
	}

	customType := strings.TrimSpace(text)
	if customType == "" {
		msg := tbapi.NewMessage(chatID, "Empty description. Use /template to try again.")
		send(msg, h.TbAPI)
		return true
	}

	if h.GenerateAndPreview != nil {
		h.GenerateAndPreview(ctx, userID, chatID, customType, customType, "")
	}
	return true
}

func (h *BotMessageHandler) handleRegenNotesInput(ctx context.Context, userID, chatID int64, text string) bool {
	h.mu.Lock()
	_, awaiting := h.awaitingRegenNotes[userID]
	if awaiting {
		delete(h.awaitingRegenNotes, userID)
	}
	h.mu.Unlock()

	if !awaiting {
		return false
	}

	notes := strings.TrimSpace(text)
	if notes == "" {
		msg := tbapi.NewMessage(chatID, "Empty notes. Use /template to try again.")
		send(msg, h.TbAPI)
		return true
	}

	if h.GenerateAndPreview != nil {
		// For regen with notes, we don't know the original type from here,
		// so we pass the notes as extra instructions with a generic type.
		h.GenerateAndPreview(ctx, userID, chatID, "custom_regen", "workout (same style as the previous attempt)", notes)
	}
	return true
}
