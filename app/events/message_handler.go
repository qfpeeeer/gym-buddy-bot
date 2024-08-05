package events

import (
	"context"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type BotMessageHandler struct {
	TbAPI TbAPI
}

func (h *BotMessageHandler) HandleMessages(ctx context.Context, update tbapi.Update) {
	userID := update.Message.From.ID
	messageText := update.Message.Text

	log.Printf("[info] received message from user %d: %s", userID, messageText)
}
