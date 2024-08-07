package events

import (
	"context"
	"fmt"
	"log"
	"strings"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotCallbackQueryHandler struct {
	TbAPI           TbAPI
	ExerciseManager ExercisesManager
	UserManager     UserManager
}

func (h *BotCallbackQueryHandler) HandleCallbackQuery(ctx context.Context, update tbapi.Update) {
	query := update.CallbackQuery
	userID := query.From.ID
	data := query.Data

	log.Printf("[info] received callback query from user %d: %s", userID, data)

	switch {
	case data == "get_exercises":
		h.handleGetExercises(query)
	case strings.HasPrefix(data, "exercise_info_"):
		h.handleExerciseInfo(query)
	case strings.HasPrefix(data, "remove_exercise_"):
		h.handleRemoveExercise(query)
	case strings.HasPrefix(data, "replace_exercise_"):
		h.handleReplaceExercise(query)
	case strings.HasPrefix(data, "back_to_exercises"):
		h.handleBackToExercises(query)
		// ... other callback queries
	}
}

func (h *BotCallbackQueryHandler) handleGetExercises(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	todayExercises := h.ExerciseManager.GetRandomExercises(5)

	if err := h.UserManager.EnsureUser(userID); err != nil {
		log.Printf("[error] failed to ensure user: %v", err)
		return
	}

	if err := h.UserManager.SetTodayExercises(userID, todayExercises); err != nil {
		log.Printf("[error] failed to set today's exercises: %v", err)
		return
	}

	h.updateExerciseMessage(query.Message.MessageID, query.Message.Chat.ID, userID)

	callback := tbapi.NewCallback(query.ID, "Exercises loaded")
	if _, err := h.TbAPI.Request(callback); err != nil {
		log.Printf("[error] failed to answer callback query: %v", err)
	}
}

func (h *BotCallbackQueryHandler) handleExerciseInfo(query *tbapi.CallbackQuery) {
	parts := strings.SplitN(query.Data, "_", 3)
	if len(parts) != 3 {
		log.Printf("[error] invalid exercise_info callback data: %s", query.Data)
		return
	}

	exerciseID := parts[2]
	exercise, found := h.ExerciseManager.GetExerciseByID(exerciseID)
	if !found {
		log.Printf("[error] exercise not found: %s", exerciseID)
		return
	}

	messageText := fmt.Sprintf("Exercise: %s\n\nCategory: %s\nForce: %s\nLevel: %s\nMechanic: %s\nEquipment: %s\n\nPrimary Muscles: %s\nSecondary Muscles: %s\n\nInstructions:\n",
		exercise.Name, exercise.Category, exercise.Force, exercise.Level, exercise.Mechanic, exercise.Equipment,
		strings.Join(exercise.PrimaryMuscles, ", "), strings.Join(exercise.SecondaryMuscles, ", "))

	for i, instruction := range exercise.Instructions {
		messageText += fmt.Sprintf("%d. %s\n", i+1, instruction)
	}

	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Replace", fmt.Sprintf("replace_exercise_%s", exerciseID)),
			tbapi.NewInlineKeyboardButtonData("Remove", fmt.Sprintf("remove_exercise_%s", exerciseID)),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Get Explanation Photos", fmt.Sprintf("get_photos_%s", exerciseID)),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Back to Exercises", "back_to_exercises"),
		),
	)

	msg := tbapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, messageText)
	msg.ReplyMarkup = &keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send exercise info message: %v", err)
	}

	callback := tbapi.NewCallback(query.ID, "Exercise info shown")
	if _, err := h.TbAPI.Request(callback); err != nil {
		log.Printf("[error] failed to answer callback query: %v", err)
	}
}

func (h *BotCallbackQueryHandler) handleRemoveExercise(query *tbapi.CallbackQuery) {
	parts := strings.SplitN(query.Data, "_", 3)
	if len(parts) != 3 {
		log.Printf("[error] invalid remove_exercise callback data: %s", query.Data)
		return
	}

	exerciseID := parts[2]
	userID := query.From.ID

	exercise, found := h.ExerciseManager.GetExerciseByID(exerciseID)
	if !found {
		log.Printf("[error] exercise not found: %s", exerciseID)

		callback := tbapi.NewCallback(query.ID, "Exercise remove failed, exercise not found")
		if _, err := h.TbAPI.Request(callback); err != nil {
			log.Printf("[error] failed to answer callback query: %v", err)
		}

		return
	}

	if err := h.UserManager.RemoveExercise(userID, exercise); err != nil {
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
	parts := strings.SplitN(query.Data, "_", 3)
	if len(parts) != 3 {
		log.Printf("[error] invalid replace_exercise callback data: %s", query.Data)
		return
	}

	exerciseID := parts[2]
	userID := query.From.ID

	oldExercise, found := h.ExerciseManager.GetExerciseByID(exerciseID)
	if !found {
		log.Printf("[error] exercise not found: %s", exerciseID)

		callback := tbapi.NewCallback(query.ID, "Exercise replace failed, exercise not found")
		if _, err := h.TbAPI.Request(callback); err != nil {
			log.Printf("[error] failed to answer callback query: %v", err)
		}

		return
	}

	newExercise := h.ExerciseManager.GetRandomExercises(1)[0]
	if err := h.UserManager.ReplaceExercise(userID, oldExercise, newExercise); err != nil {
		log.Printf("[error] failed to replace exercise: %v", err)
		return
	}

	// Update the message with the new exercise information
	messageText := fmt.Sprintf("Exercise: %s\n\nCategory: %s\nForce: %s\nLevel: %s\nMechanic: %s\nEquipment: %s\n\nPrimary Muscles: %s\nSecondary Muscles: %s\n\nInstructions:\n",
		newExercise.Name, newExercise.Category, newExercise.Force, newExercise.Level, newExercise.Mechanic, newExercise.Equipment,
		strings.Join(newExercise.PrimaryMuscles, ", "), strings.Join(newExercise.SecondaryMuscles, ", "))

	for i, instruction := range newExercise.Instructions {
		messageText += fmt.Sprintf("%d. %s\n", i+1, instruction)
	}

	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Replace", fmt.Sprintf("replace_exercise_%s", newExercise.ID)),
			tbapi.NewInlineKeyboardButtonData("Remove", fmt.Sprintf("remove_exercise_%s", newExercise.ID)),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Get ExplanationЫ Photos", fmt.Sprintf("get_photos_%s", newExercise.ID)),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Back to Exercises", "back_to_exercises"),
		),
	)

	msg := tbapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, messageText)
	msg.ReplyMarkup = &keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to update message with new exercise: %v", err)
	}

	callback := tbapi.NewCallback(query.ID, "Exercise replaced")
	if _, err := h.TbAPI.Request(callback); err != nil {
		log.Printf("[error] failed to answer callback query: %v", err)
	}
}

func (h *BotCallbackQueryHandler) handleBackToExercises(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	h.updateExerciseMessage(query.Message.MessageID, query.Message.Chat.ID, userID)

	callback := tbapi.NewCallback(query.ID, "Back to exercises")
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

	messageText := "Here are your exercises for today:\n\n"
	var keyboardRows [][]tbapi.InlineKeyboardButton

	for i, exercise := range exercises {
		messageText += fmt.Sprintf("%d. %s (%s)\n", i+1, exercise.Name, exercise.Category)
		button := tbapi.NewInlineKeyboardButtonData(exercise.Name, fmt.Sprintf("exercise_info_%s", exercise.ID))
		keyboardRows = append(keyboardRows, []tbapi.InlineKeyboardButton{button})
	}

	msg := tbapi.NewEditMessageText(chatID, messageID, messageText)
	keyboard := tbapi.NewInlineKeyboardMarkup(keyboardRows...)
	msg.ReplyMarkup = &keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to update exercises message: %v", err)
	}
}
