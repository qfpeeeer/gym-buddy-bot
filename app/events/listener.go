package events

import (
	"context"
	"fmt"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type TelegramListener struct {
	TbAPI                TbAPI
	MessageHandler       MessageHandler
	CommandHandler       CommandHandler
	CallbackQueryHandler CallbackQueryHandler
}

func (l *TelegramListener) StartListening(ctx context.Context) error {
	log.Printf("[info] started telegram bot")

	u := tbapi.NewUpdate(0)
	u.Timeout = 60

	updates := l.TbAPI.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case update, ok := <-updates:
			if !ok {
				return fmt.Errorf("telegram updates channel closed")
			}

			if update.Message != nil {
				if update.Message.IsCommand() {
					l.CommandHandler.HandleCommands(ctx, update)
				} else {
					l.MessageHandler.HandleMessages(ctx, update)
				}
			} else if update.CallbackQuery != nil {
				l.CallbackQueryHandler.HandleCallbackQuery(ctx, update)
			}
		}
	}
}
