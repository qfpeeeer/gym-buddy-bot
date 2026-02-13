package events

import (
	"context"
	"log"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotCallbackQueryHandler struct {
	TbAPI              TbAPI
	UserManager        UserManager
	SetAwaitingHevyKey func(userID, chatID int64)
}

func (h *BotCallbackQueryHandler) HandleCallbackQuery(ctx context.Context, update tbapi.Update) {
	query := update.CallbackQuery
	userID := query.From.ID
	data := query.Data

	log.Printf("[info] received callback query from user %d: %s", userID, data)

	switch data {
	case "connect_hevy":
		h.handleConnectHevy(query)
	}
}

func (h *BotCallbackQueryHandler) handleConnectHevy(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "")
	if _, err := h.TbAPI.Request(callback); err != nil {
		log.Printf("[error] failed to answer callback query: %v", err)
	}

	text := "Send me your Hevy API key.\n\nGet it from: hevy.com/settings > Developer section.\nRequires Hevy Pro subscription."
	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)

	if h.SetAwaitingHevyKey != nil {
		h.SetAwaitingHevyKey(userID, chatID)
	}
}
