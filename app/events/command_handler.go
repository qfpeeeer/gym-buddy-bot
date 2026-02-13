package events

import (
	"context"
	"log"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotCommandHandler struct {
	TbAPI              TbAPI
	UserManager        UserManager
	SetAwaitingHevyKey func(userID, chatID int64)
}

func (h *BotCommandHandler) HandleCommands(ctx context.Context, update tbapi.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	command := update.Message.Command()

	log.Printf("[info] received command from user %d: %s", userID, command)

	if err := h.UserManager.EnsureUser(userID); err != nil {
		log.Printf("[error] failed to ensure user: %v", err)
	}

	switch command {
	case "start":
		h.handleStart(chatID, userID)
	case "connect":
		h.handleConnect(chatID, userID)
	case "disconnect":
		h.handleDisconnect(chatID, userID)
	}
}

func (h *BotCommandHandler) handleStart(chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)

	var rows [][]tbapi.InlineKeyboardButton

	if !connected {
		rows = append(rows, tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Connect Hevy Account", "connect_hevy"),
		))
	} else {
		rows = append(rows, tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Fetch Latest Workout", "fetch_last"),
		))
	}

	keyboard := tbapi.NewInlineKeyboardMarkup(rows...)

	text := "Welcome to GymBuddy!\n\n"
	if connected {
		text += "Your Hevy account is connected.\nUse /sync to import new workouts or /last to see your latest session."
	} else {
		text += "Connect your Hevy account with /connect to unlock workout sync and AI analysis."
	}

	msg := tbapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send start message: %v", err)
	}
}

func (h *BotCommandHandler) handleConnect(chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)
	if connected {
		msg := tbapi.NewMessage(chatID, "Hevy account is already connected. Use /disconnect first to reconnect with a different key.")
		send(msg, h.TbAPI)
		return
	}

	text := "Send me your Hevy API key.\n\nGet it from: hevy.com/settings > Developer section.\nRequires Hevy Pro subscription."
	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)

	if h.SetAwaitingHevyKey != nil {
		h.SetAwaitingHevyKey(userID, chatID)
	}
}

func (h *BotCommandHandler) handleDisconnect(chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)
	if !connected {
		msg := tbapi.NewMessage(chatID, "No Hevy account connected.")
		send(msg, h.TbAPI)
		return
	}

	if err := h.UserManager.ClearHevyAPIKey(userID); err != nil {
		log.Printf("[error] failed to clear hevy key: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to disconnect. Please try again.")
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, "Hevy account disconnected.")
	send(msg, h.TbAPI)
}
