package events

import (
	"context"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type BotCommandHandler struct {
	TbAPI           TbAPI
	ExerciseManager ExercisesManager
}

func (h *BotCommandHandler) HandleCommands(ctx context.Context, update tbapi.Update) {
	userID := update.Message.From.ID
	command := update.Message.Command()

	log.Printf("[info] received command from user %d: %s", userID, command)

	switch command {
	case "start":
		h.handleStart(update.Message.Chat.ID)
		// ... other commands
	}
}

func (h *BotCommandHandler) handleStart(chatID int64) {
	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Get today's exercises", "get_exercises"),
		),
	)

	msg := tbapi.NewMessage(chatID, "Welcome to GymBuddy! What would you like to do?")
	msg.ReplyMarkup = keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send start message: %v", err)
	}
}
