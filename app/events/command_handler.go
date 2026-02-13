package events

import (
	"context"
	"fmt"
	"log"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotCommandHandler struct {
	TbAPI           TbAPI
	ExerciseManager ExercisesManager
	UserManager     UserManager
}

func (h *BotCommandHandler) HandleCommands(ctx context.Context, update tbapi.Update) {
	userID := update.Message.From.ID
	command := update.Message.Command()

	log.Printf("[info] received command from user %d: %s", userID, command)

	switch command {
	case "start":
		h.handleStart(update.Message.Chat.ID)
	case "history":
		h.handleHistory(update.Message.Chat.ID, userID)
	}
}

func (h *BotCommandHandler) handleStart(chatID int64) {
	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Get today's exercises", "get_exercises"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Workout history", "get_history"),
		),
	)

	text := "Welcome to GymBuddy!\n\nPaste a workout from Strong or upload a CSV export to save your training history."
	msg := tbapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send start message: %v", err)
	}
}

func (h *BotCommandHandler) handleHistory(chatID int64, userID int64) {
	workouts, err := h.UserManager.GetRecentWorkouts(userID, 5)
	if err != nil {
		log.Printf("[error] failed to get recent workouts: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to load workout history.")
		send(msg, h.TbAPI)
		return
	}

	if len(workouts) == 0 {
		msg := tbapi.NewMessage(chatID, "No workouts logged yet.\n\nPaste a workout from the Strong app or upload a CSV export to get started.")
		send(msg, h.TbAPI)
		return
	}

	text := "Recent workouts:\n"
	for i, w := range workouts {
		totalSets := 0
		for _, ex := range w.Exercises {
			totalSets += len(ex.Sets)
		}
		text += fmt.Sprintf("\n%d. %s\n   %s | %d exercise(s), %d set(s)",
			i+1,
			w.Name,
			w.Date.Format("Mon, 2 Jan 2006"),
			len(w.Exercises),
			totalSets,
		)
	}

	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)
}
