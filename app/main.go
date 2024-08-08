package main

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/qfpeeeer/gym-buddy-bot/app/events"
	"github.com/qfpeeeer/gym-buddy-bot/app/services/exercises"
	services "github.com/qfpeeeer/gym-buddy-bot/app/services/googleSheets"
	"github.com/qfpeeeer/gym-buddy-bot/app/services/user"
	"github.com/qfpeeeer/gym-buddy-bot/app/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	environment := os.Getenv("ENVIRONMENT")

	fmt.Printf("gym-buddy-bot %s\n", environment)
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

	googleSheetsClientID := os.Getenv("GOOGLE_SHEETS_CLIENT_ID")
	googleSheetsClientSecret := os.Getenv("GOOGLE_SHEETS_CLIENT_SECRET")
	googleSheetsRedirectURL := "http://localhost:8080/oauth2callback" // TODO: change to the actual redirect URL

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

	googleSheetsStorage, err := storage.NewGoogleSheetsStorage(dataDB)
	if err != nil {
		return fmt.Errorf("failed to initialize google sheets storage: %w", err)
	}

	userStateStorage, err := storage.NewUserStateStorage(dataDB)
	if err != nil {
		return fmt.Errorf("failed to initialize user state storage: %w", err)
	}

	userManager := user.NewManager(userStorage, exerciseStorage, googleSheetsStorage, userStateStorage)

	tbAPI, err := tbapi.NewBotAPI(telegramToken)
	if err != nil {
		return fmt.Errorf("can't make telegram bot, %w", err)
	}
	tbAPI.Debug = false

	exercisesManager, err := exercises.NewExerciseManager("exercises.json")
	if err != nil {
		return fmt.Errorf("can't make exercises db, %w", err)
	}

	googleSheetsService := services.NewGoogleSheetsService(
		googleSheetsClientID,
		googleSheetsClientSecret,
		googleSheetsRedirectURL,
		userManager,
		tbAPI,
	)

	commandHandler := &events.BotCommandHandler{
		TbAPI:               tbAPI,
		ExerciseManager:     exercisesManager,
		UserManager:         userManager,
		GoogleSheetsService: googleSheetsService,
	}

	messageHandler := &events.BotMessageHandler{
		TbAPI:       tbAPI,
		UserManager: userManager,
	}

	callbackQueryHandler := &events.BotCallbackQueryHandler{
		TbAPI:               tbAPI,
		ExerciseManager:     exercisesManager,
		UserManager:         userManager,
		GoogleSheetsService: googleSheetsService,
	}

	listener := events.TelegramListener{
		TbAPI:                tbAPI,
		CommandHandler:       commandHandler,
		MessageHandler:       messageHandler,
		CallbackQueryHandler: callbackQueryHandler,
	}

	http.HandleFunc("/oauth2callback", googleSheetsService.HandleRedirect)
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	err = listener.StartListening(ctx)
	if err != nil {
		return fmt.Errorf("failed to start listening: %w", err)
	}

	return nil
}
