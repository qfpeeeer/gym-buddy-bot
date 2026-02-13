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

type awaitingKeyState struct {
	chatID int64
}

type BotMessageHandler struct {
	TbAPI       TbAPI
	UserManager UserManager

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
}

// handleHevyKeyInput checks if the user is awaiting API key input and processes it.
// Returns true if the message was handled as a key input.
func (h *BotMessageHandler) handleHevyKeyInput(ctx context.Context, userID, chatID int64, messageID int, text string) bool {
	h.mu.Lock()
	_, awaiting := h.awaitingKey[userID]
	if awaiting {
		delete(h.awaitingKey, userID)
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
