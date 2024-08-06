package main

import (
	"context"
	"fmt"
	"github.com/qfpeeeer/gym-buddy-bot/app/events"
	"github.com/qfpeeeer/gym-buddy-bot/app/exercises"
	"log"
	"os"
	"os/signal"
	"syscall"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	revision := os.Getenv("REVISION")

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

	exercisesDB, err := exercises.NewExerciseDB("exercises.json")
	if err != nil {
		return fmt.Errorf("can't make exercises db, %w", err)
	}

	commandHandler := &events.BotCommandHandler{
		TbAPI:      tbAPI,
		ExerciseDB: exercisesDB,
	}

	messageHandler := &events.BotMessageHandler{
		TbAPI: tbAPI,
	}

	callbackQueryHandler := &events.BotCallbackQueryHandler{
		TbAPI:      tbAPI,
		ExerciseDB: exercisesDB,
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
