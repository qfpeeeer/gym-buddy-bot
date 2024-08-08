package events

import (
	"context"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type BotCommandHandler struct {
	TbAPI               TbAPI
	ExerciseManager     ExercisesManager
	UserManager         UserManager
	GoogleSheetsService GoogleSheetsService
}

func (h *BotCommandHandler) HandleCommands(ctx context.Context, update tbapi.Update) {
	userID := update.Message.From.ID
	command := update.Message.Command()

	log.Printf("[info] received command from user %d: %s", userID, command)

	switch command {
	case "start":
		h.handleStart(update.Message.Chat.ID)
	case "connect_sheets":
		h.handleConnectSheets(update.Message.Chat.ID, userID)
	case "cancel":
		h.handleCancel(update.Message.Chat.ID, userID)
	}
}

func (h *BotCommandHandler) handleStart(chatID int64) {
	// Send main menu message
	h.sendMainMenu(chatID)
}

func (h *BotCommandHandler) handleCancel(chatID int64, userID int64) {
	// Reset user state
	err := h.UserManager.SetUserState(userID, "")
	if err != nil {
		log.Printf("[error] failed to reset state for user %d: %v", userID, err)
		h.sendErrorMessage(chatID, "An error occurred. Please try again later.")
		return
	}

	// Send main menu message
	h.sendMainMenu(chatID)
}

func (h *BotCommandHandler) sendMainMenu(chatID int64) {
	message := "Welcome to the GymBuddy Bot! Here are the available commands:\n\n" +
		"/connect_sheets - Connect or manage your Google Sheets\n" +
		"/help - Show available commands\n" +
		"/cancel - Cancel current operation and return to main menu"

	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Get today's exercises", "get_exercises"),
		),
	)

	msg := tbapi.NewMessage(chatID, message)
	msg.ReplyMarkup = keyboard
	_, err := h.TbAPI.Send(msg)
	if err != nil {
		log.Printf("[error] failed to send main menu message: %v", err)
	}
}

func (h *BotCommandHandler) handleConnectSheets(chatID int64, userID int64) {
	// Check if the user is already connected
	token, err := h.UserManager.GetGoogleSheetsToken(userID)
	if err == nil && token != nil {
		// User is already connected
		keyboard := tbapi.NewInlineKeyboardMarkup(
			tbapi.NewInlineKeyboardRow(
				tbapi.NewInlineKeyboardButtonData("Reconnect", "reconnect_sheets"),
				tbapi.NewInlineKeyboardButtonData("Change Sheet", "change_sheet"),
			),
		)

		msg := tbapi.NewMessage(chatID, "You're already connected to Google Sheets. What would you like to do?")
		msg.ReplyMarkup = keyboard
		if _, err := h.TbAPI.Send(msg); err != nil {
			log.Printf("[error] failed to send start message: %v", err)
		}
		return
	}

	// Generate the OAuth URL
	authURL, err := h.GoogleSheetsService.GetAuthorizationURL(userID)
	if err != nil {
		log.Printf("[error] failed to generate authorization URL: %v", err)
		h.sendErrorMessage(chatID, "Sorry, there was an error generating the authorization URL. Please try again later.")
		return
	}

	// Create an inline keyboard with the authorization link
	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonURL("Authorize Google Sheets", authURL),
		),
	)

	msg := tbapi.NewMessage(chatID, "Please click the button below to authorize access to your Google Sheets. After authorization, you'll be redirected to a page with further instructions.")
	msg.ReplyMarkup = keyboard
	if _, err = h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send start message: %v", err)
	}

	// Set user state to waiting for auth
	err = h.UserManager.SetUserState(userID, "waiting_for_auth")
	if err != nil {
		log.Printf("[error] failed to set user state: %v", err)
	}
}

func (h *BotCommandHandler) sendErrorMessage(chatID int64, text string) {
	msg := tbapi.NewMessage(chatID, text)
	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send start message: %v", err)
	}
}
