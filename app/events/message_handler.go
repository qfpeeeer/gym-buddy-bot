package events

import (
	"context"
	"fmt"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
)

type BotMessageHandler struct {
	TbAPI       TbAPI
	UserManager UserManager
}

func (h *BotMessageHandler) HandleMessages(ctx context.Context, update tbapi.Update) {
	userID := update.Message.From.ID
	messageText := update.Message.Text

	log.Printf("[info] received message from user %d: %s", userID, messageText)

	// Check the user's current state
	state, err := h.UserManager.GetUserState(userID)
	if err != nil {
		log.Printf("[error] failed to get user state: %v", err)
		h.sendErrorMessage(update.Message.Chat.ID, "An error occurred. Please try again later.")
		return
	}

	switch state {
	case "waiting_for_sheet_id":
		h.handleSheetID(update.Message.Chat.ID, userID, messageText)
	default:
		h.sendMessage(update.Message.Chat.ID, "I'm not sure how to process that message. You can use /connect to start connecting your Google Sheet.")
	}
}

func (h *BotMessageHandler) handleSheetID(chatID int64, userID int64, sheetID string) {
	// Basic validation of the Sheet ID
	sheetID = strings.TrimSpace(sheetID)
	if !strings.HasPrefix(sheetID, "1") || len(sheetID) != 44 {
		h.sendMessage(chatID, "That doesn't look like a valid Google Sheet ID. Please try again. The ID should be 44 characters long and start with '1'.")
		return
	}

	// Save the Sheet ID
	err := h.UserManager.StoreGoogleSheetID(userID, sheetID)
	if err != nil {
		log.Printf("[error] failed to store Sheet ID for user %d: %v", userID, err)
		h.sendErrorMessage(chatID, "An error occurred while saving your Sheet ID. Please try again later.")
		return
	}

	// Reset the user's state
	err = h.UserManager.SetUserState(userID, "")
	if err != nil {
		log.Printf("[error] failed to reset state for user %d: %v", userID, err)
	}

	// Send success message
	h.sendMessage(chatID, fmt.Sprintf("Great! Your Google Sheet (ID: %s) has been successfully connected. You can now use the bot to write to your sheet.", sheetID))
}

func (h *BotMessageHandler) sendMessage(chatID int64, text string) {
	msg := tbapi.NewMessage(chatID, text)
	_, err := h.TbAPI.Send(msg)
	if err != nil {
		log.Printf("[error] failed to send message: %v", err)
	}
}

func (h *BotMessageHandler) sendErrorMessage(chatID int64, text string) {
	h.sendMessage(chatID, text)
}
