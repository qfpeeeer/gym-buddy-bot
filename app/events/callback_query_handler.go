package events

import (
	"context"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type BotCallbackQueryHandler struct {
	TbAPI TbAPI
}

func (h *BotCallbackQueryHandler) HandleCallbackQuery(ctx context.Context, update tbapi.Update) {
	userID := update.CallbackQuery.From.ID
	data := update.CallbackQuery.Data

	log.Printf("[info] received callback query from user %d: %s", userID, data)
}
