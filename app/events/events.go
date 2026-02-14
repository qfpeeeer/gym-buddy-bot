package events

import (
	"context"
	"fmt"
	"log"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
	"github.com/qfpeeeer/gym-buddy-bot/app/storage"
)

// TbAPI is an interface for telegram bot API, only subset of methods used
type TbAPI interface {
	GetUpdatesChan(config tbapi.UpdateConfig) tbapi.UpdatesChannel
	Send(c tbapi.Chattable) (tbapi.Message, error)
	Request(c tbapi.Chattable) (*tbapi.APIResponse, error)
	GetChat(config tbapi.ChatInfoConfig) (tbapi.Chat, error)
}

type CommandHandler interface {
	HandleCommands(ctx context.Context, update tbapi.Update)
}

type MessageHandler interface {
	HandleMessages(ctx context.Context, update tbapi.Update)
}

type CallbackQueryHandler interface {
	HandleCallbackQuery(ctx context.Context, update tbapi.Update)
}

type UserManager interface {
	EnsureUser(telegramID int64) error
	SetHevyAPIKey(telegramID int64, key string) error
	GetHevyAPIKey(telegramID int64) (string, error)
	ClearHevyAPIKey(telegramID int64) error
	IsHevyConnected(telegramID int64) (bool, error)
	IsSynced(userID int64) (bool, error)
	InitSync(ctx context.Context, userID int64, progress func(string)) error
	IncrementalSync(ctx context.Context, userID int64) (int, error)
	GetLastWorkout(ctx context.Context, userID int64) (*hevy.Workout, error)
	GetWorkoutsByTitle(userID int64, title string) ([]hevy.Workout, error)
	RunAnalysis(userID int64, reportType string) (string, error)

	// AI settings
	SetOpenAIKey(userID int64, key string) error
	SetAIModel(userID int64, model string) error
	GetAISettings(userID int64) (*storage.AISettings, error)
	IsAIConfigured(userID int64) (bool, error)

	// User preferences
	SetGoals(userID int64, goals string) error
	SetNotes(userID int64, notes string) error
	GetPreferences(userID int64) (*storage.UserPreferences, error)

	// AI advice
	GetAdvice(ctx context.Context, userID int64) (string, error)
	AskQuestion(ctx context.Context, userID int64, question string) (string, error)
	GetAnalysisInsights(ctx context.Context, userID int64, analysisText string) (string, error)
}

// send a message to the telegram as markdown first and if failed - as plain text
func send(tbMsg tbapi.Chattable, tbAPI TbAPI) error {
	withParseMode := func(tbMsg tbapi.Chattable, parseMode string) tbapi.Chattable {
		switch msg := tbMsg.(type) {
		case tbapi.MessageConfig:
			msg.ParseMode = parseMode
			msg.DisableWebPagePreview = true
			return msg
		case tbapi.EditMessageTextConfig:
			msg.ParseMode = parseMode
			msg.DisableWebPagePreview = true
			return msg
		case tbapi.EditMessageReplyMarkupConfig:
			return msg
		}
		return tbMsg
	}

	msg := withParseMode(tbMsg, tbapi.ModeMarkdown) // try markdown first
	if _, err := tbAPI.Send(msg); err != nil {
		log.Printf("[warn] failed to send message as markdown, %v", err)
		msg = withParseMode(tbMsg, "") // try plain text
		if _, err := tbAPI.Send(msg); err != nil {
			return fmt.Errorf("can't send message to telegram: %w", err)
		}
	}

	return nil
}
