package events

import (
	"context"
	"fmt"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qfpeeeer/gym-buddy-bot/app/services/exercises"
	"golang.org/x/oauth2"
	"log"
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
	SetTodayExercises(telegramID int64, exercises []exercises.Exercise) error
	GetTodayExercises(telegramID int64) ([]exercises.Exercise, error)
	RemoveExercise(telegramID int64, exercise exercises.Exercise) error
	ReplaceExercise(telegramID int64, oldExercise, newExercise exercises.Exercise) error
	StoreGoogleSheetID(telegramID int64, sheetID string) error
	SetUserState(telegramID int64, state string) error
	GetUserState(telegramID int64) (string, error)
	GetGoogleSheetsToken(telegramID int64) (*oauth2.Token, error)
	GetGoogleSheetID(telegramID int64) (string, error)
}

type ExercisesManager interface {
	GetRandomExercises(count int) []exercises.Exercise
	GetExerciseByID(id string) (exercises.Exercise, bool)
}

type GoogleSheetsService interface {
	GetAuthorizationURL(userID int64) (string, error)
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
