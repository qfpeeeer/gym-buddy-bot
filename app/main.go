package main

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/qfpeeeer/gym-buddy-bot/app/events"
	"github.com/qfpeeeer/gym-buddy-bot/app/exercises"
	"github.com/qfpeeeer/gym-buddy-bot/app/storage"
	"github.com/qfpeeeer/gym-buddy-bot/app/user"
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
	dataFilePath := os.Getenv("DATA_FILE_PATH")
	telegramToken := os.Getenv("TELEGRAM_TOKEN")

	dataDB, err := storage.NewSqliteDB(dataFilePath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %v", err)
	}
	defer func(dataDB *sqlx.DB) {
		err = dataDB.Close()
		if err != nil {
			log.Printf("[warn] error closing sqlite database: %v", err)
		}
	}(dataDB)

	userStorage, err := storage.NewUserStorage(dataDB)
	if err != nil {
		return fmt.Errorf("failed to initialize user storage: %w", err)
	}

	exerciseStorage, err := storage.NewExerciseStorage(dataDB)
	if err != nil {
		return fmt.Errorf("failed to initialize exercise storage: %w", err)
	}

	userManager := user.NewManager(userStorage, exerciseStorage)

	tbAPI, err := tbapi.NewBotAPI(telegramToken)
	if err != nil {
		return fmt.Errorf("can't make telegram bot, %w", err)
	}
	tbAPI.Debug = false

	exercisesManager, err := exercises.NewExerciseManager("exercises.json")
	if err != nil {
		return fmt.Errorf("can't make exercises db, %w", err)
	}

	commandHandler := &events.BotCommandHandler{
		TbAPI:      tbAPI,
		ExerciseDB: exercisesManager,
	}

	messageHandler := &events.BotMessageHandler{
		TbAPI: tbAPI,
	}

	callbackQueryHandler := &events.BotCallbackQueryHandler{
		TbAPI:       tbAPI,
		ExerciseDB:  exercisesManager,
		UserManager: userManager,
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
