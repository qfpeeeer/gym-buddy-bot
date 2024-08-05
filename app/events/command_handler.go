package events

import (
	"context"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type BotCommandHandler struct {
	TbAPI TbAPI
}

func (h *BotCommandHandler) HandleCommands(ctx context.Context, update tbapi.Update) {
	userID := update.Message.From.ID
	command := update.Message.Command()

	log.Printf("[info] received command from user %d: %s", userID, command)
}
