package events

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotCallbackQueryHandler struct {
	TbAPI       TbAPI
	ExerciseDB  ExercisesManager
	UserManager UserManager
}

func (h *BotCallbackQueryHandler) HandleCallbackQuery(ctx context.Context, update tbapi.Update) {
	query := update.CallbackQuery
	userID := query.From.ID
	data := query.Data

	log.Printf("[info] received callback query from user %d: %s", userID, data)

	switch {
	case data == "get_exercises":
		h.handleGetExercises(query)
	case strings.HasPrefix(data, "remove_exercise_"):
		h.handleRemoveExercise(query)
	case strings.HasPrefix(data, "replace_exercise_"):
		h.handleReplaceExercise(query)
		// ... other callback queries
	}
}

func (h *BotCallbackQueryHandler) handleGetExercises(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	todayExercises := h.ExerciseDB.GetRandomExercises(5)

	if err := h.UserManager.EnsureUser(userID); err != nil {
		log.Printf("[error] failed to ensure user: %v", err)
		return
	}

	if err := h.UserManager.SetTodayExercises(userID, todayExercises); err != nil {
		log.Printf("[error] failed to set today's exercises: %v", err)
		return
	}

	messageText := "Here are your exercises for today:\n\n"
	var keyboardRows [][]tbapi.InlineKeyboardButton

	for i, exercise := range todayExercises {
		messageText += fmt.Sprintf("%d. %s (%s)\n", i+1, exercise.Name, exercise.Category)

		removeButton := tbapi.NewInlineKeyboardButtonData(fmt.Sprintf("Remove %d", i+1), fmt.Sprintf("remove_exercise_%d", i))
		replaceButton := tbapi.NewInlineKeyboardButtonData(fmt.Sprintf("Replace %d", i+1), fmt.Sprintf("replace_exercise_%d", i))
		keyboardRows = append(keyboardRows, []tbapi.InlineKeyboardButton{removeButton, replaceButton})
	}

	msg := tbapi.NewMessage(query.Message.Chat.ID, messageText)
	msg.ReplyMarkup = tbapi.NewInlineKeyboardMarkup(keyboardRows...)

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send exercises message: %v", err)
	}

	callback := tbapi.NewCallback(query.ID, "Exercises loaded")
	if _, err := h.TbAPI.Request(callback); err != nil {
		log.Printf("[error] failed to answer callback query: %v", err)
	}
}

func (h *BotCallbackQueryHandler) handleRemoveExercise(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		log.Printf("[error] invalid remove_exercise callback data: %s", query.Data)
		return
	}

	index, err := strconv.Atoi(parts[2])
	if err != nil {
		log.Printf("[error] invalid exercise index: %v", err)
		return
	}

	if err := h.UserManager.RemoveExercise(userID, index); err != nil {
		log.Printf("[error] failed to remove exercise: %v", err)
		return
	}

	h.updateExerciseMessage(query.Message.MessageID, query.Message.Chat.ID, userID)

	callback := tbapi.NewCallback(query.ID, "Exercise removed")
	if _, err := h.TbAPI.Request(callback); err != nil {
		log.Printf("[error] failed to answer callback query: %v", err)
	}
}

func (h *BotCallbackQueryHandler) handleReplaceExercise(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	parts := strings.Split(query.Data, "_")
	if len(parts) != 3 {
		log.Printf("[error] invalid replace_exercise callback data: %s", query.Data)
		return
	}

	index, err := strconv.Atoi(parts[2])
	if err != nil {
		log.Printf("[error] invalid exercise index: %v", err)
		return
	}

	newExercise := h.ExerciseDB.GetRandomExercises(1)[0]
	if err := h.UserManager.ReplaceExercise(userID, index, newExercise); err != nil {
		log.Printf("[error] failed to replace exercise: %v", err)
		return
	}

	h.updateExerciseMessage(query.Message.MessageID, query.Message.Chat.ID, userID)

	callback := tbapi.NewCallback(query.ID, "Exercise replaced")
	if _, err := h.TbAPI.Request(callback); err != nil {
		log.Printf("[error] failed to answer callback query: %v", err)
	}
}

func (h *BotCallbackQueryHandler) updateExerciseMessage(messageID int, chatID int64, userID int64) {
	exercises, err := h.UserManager.GetTodayExercises(userID)
	if err != nil {
		log.Printf("[error] failed to get today's exercises: %v", err)
		return
	}

	messageText := "Here are your updated exercises for today:\n\n"
	var keyboardRows [][]tbapi.InlineKeyboardButton

	for i, exercise := range exercises {
		messageText += fmt.Sprintf("%d. %s (%s)\n", i+1, exercise.Name, exercise.Category)

		removeButton := tbapi.NewInlineKeyboardButtonData(fmt.Sprintf("Remove %d", i+1), fmt.Sprintf("remove_exercise_%d", i))
		replaceButton := tbapi.NewInlineKeyboardButtonData(fmt.Sprintf("Replace %d", i+1), fmt.Sprintf("replace_exercise_%d", i))
		keyboardRows = append(keyboardRows, []tbapi.InlineKeyboardButton{removeButton, replaceButton})
	}

	msg := tbapi.NewEditMessageText(chatID, messageID, messageText)
	keyboard := tbapi.NewInlineKeyboardMarkup(keyboardRows...)
	msg.ReplyMarkup = &keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to update exercises message: %v", err)
	}
}
