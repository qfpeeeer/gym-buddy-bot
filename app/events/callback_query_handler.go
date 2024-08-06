package events

import (
	"context"
	"fmt"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qfpeeeer/gym-buddy-bot/app/exercises"
	"log"
)

type BotCallbackQueryHandler struct {
	TbAPI      TbAPI
	ExerciseDB *exercises.ExerciseDB
}

func (h *BotCallbackQueryHandler) HandleCallbackQuery(ctx context.Context, update tbapi.Update) {
	query := update.CallbackQuery
	userID := query.From.ID
	data := query.Data

	log.Printf("[info] received callback query from user %d: %s", userID, data)

	switch data {
	case "get_exercises":
		h.handleGetExercises(query)
	}
}

func (h *BotCallbackQueryHandler) handleGetExercises(query *tbapi.CallbackQuery) {
	// Get today's exercises (for now, let's just get 5 random exercises)
	todayExercises := h.ExerciseDB.GetRandomExercises(5)

	var messageText string
	for i, exercise := range todayExercises {
		messageText += fmt.Sprintf("%d. %s (%s)\n", i+1, exercise.Name, exercise.Category)
	}

	msg := tbapi.NewMessage(query.Message.Chat.ID, "Here are your exercises for today:\n\n"+messageText)

	// Add buttons for each exercise
	var keyboardRows [][]tbapi.InlineKeyboardButton
	for _, exercise := range todayExercises {
		button := tbapi.NewInlineKeyboardButtonData(exercise.Name, "exercise_"+exercise.Name)
		keyboardRows = append(keyboardRows, []tbapi.InlineKeyboardButton{button})
	}
	msg.ReplyMarkup = tbapi.NewInlineKeyboardMarkup(keyboardRows...)

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send exercises message: %v", err)
	}

	// Answer the callback query to remove the loading indicator
	callback := tbapi.NewCallback(query.ID, "")
	if _, err := h.TbAPI.Request(callback); err != nil {
		log.Printf("[error] failed to answer callback query: %v", err)
	}
}
