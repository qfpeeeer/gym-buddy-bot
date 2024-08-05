package main

import (
	"context"
	"fmt"
	"github.com/qfpeeeer/gym-buddy-bot/app/events"
	"log"
	"os"
	"os/signal"
	"syscall"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var revision = "local"

func main() {
	fmt.Printf("gym-buddy-bot %s\n", revision)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Printf("[warn] interrupt signal")
		cancel()
	}()

	if err := execute(ctx); err != nil {
		log.Printf("[error] %v", err)
		os.Exit(1)
	}
}

func execute(ctx context.Context) error {
	telegramToken := os.Getenv("TELEGRAM_TOKEN")

	tbAPI, err := tbapi.NewBotAPI(telegramToken)
	if err != nil {
		return fmt.Errorf("can't make telegram bot, %w", err)
	}
	tbAPI.Debug = false

	commandHandler := &events.BotCommandHandler{
		TbAPI: tbAPI,
	}

	messageHandler := &events.BotMessageHandler{
		TbAPI: tbAPI,
	}

	callbackQueryHandler := &events.BotCallbackQueryHandler{
		TbAPI: tbAPI,
	}

	listener := events.TelegramListener{
		TbAPI:                tbAPI,
		CommandHandler:       commandHandler,
		MessageHandler:       messageHandler,
		CallbackQueryHandler: callbackQueryHandler,
	}

	err = listener.StartListening(ctx)
	if err != nil {
		return fmt.Errorf("failed to start listening: %w", err)
	}

	return nil
}
