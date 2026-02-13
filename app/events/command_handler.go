package events

import (
	"context"
	"fmt"
	"log"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotCommandHandler struct {
	TbAPI              TbAPI
	ExerciseManager    ExercisesManager
	UserManager        UserManager
	SetAwaitingHevyKey func(userID, chatID int64)
}

func (h *BotCommandHandler) HandleCommands(ctx context.Context, update tbapi.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	command := update.Message.Command()

	log.Printf("[info] received command from user %d: %s", userID, command)

	if err := h.UserManager.EnsureUser(userID); err != nil {
		log.Printf("[error] failed to ensure user: %v", err)
	}

	switch command {
	case "start":
		h.handleStart(chatID, userID)
	case "history":
		h.handleHistory(chatID, userID)
	case "connect":
		h.handleConnect(chatID, userID)
	case "disconnect":
		h.handleDisconnect(chatID, userID)
	}
}

func (h *BotCommandHandler) handleStart(chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)

	var rows [][]tbapi.InlineKeyboardButton

	if !connected {
		rows = append(rows, tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Connect Hevy Account", "connect_hevy"),
		))
	} else {
		rows = append(rows, tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Fetch Latest Workout", "fetch_last"),
		))
	}

	rows = append(rows,
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Get today's exercises", "get_exercises"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Workout history", "get_history"),
		),
	)

	keyboard := tbapi.NewInlineKeyboardMarkup(rows...)

	text := "Welcome to GymBuddy!\n\n"
	if connected {
		text += "Your Hevy account is connected.\nUse /sync to import new workouts or /last to see your latest session."
	} else {
		text += "Connect your Hevy account with /connect to unlock workout sync and AI analysis.\n\nYou can also paste a workout from Strong or upload a CSV export."
	}

	msg := tbapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send start message: %v", err)
	}
}

func (h *BotCommandHandler) handleConnect(chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)
	if connected {
		msg := tbapi.NewMessage(chatID, "Hevy account is already connected. Use /disconnect first to reconnect with a different key.")
		send(msg, h.TbAPI)
		return
	}

	text := "Send me your Hevy API key.\n\nGet it from: hevy.com/settings > Developer section.\nRequires Hevy Pro subscription."
	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)

	// The actual key handling happens in MessageHandler via the awaitingHevyKey state.
	if h.SetAwaitingHevyKey != nil {
		h.SetAwaitingHevyKey(userID, chatID)
	}
}

func (h *BotCommandHandler) handleDisconnect(chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)
	if !connected {
		msg := tbapi.NewMessage(chatID, "No Hevy account connected.")
		send(msg, h.TbAPI)
		return
	}

	if err := h.UserManager.ClearHevyAPIKey(userID); err != nil {
		log.Printf("[error] failed to clear hevy key: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to disconnect. Please try again.")
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, "Hevy account disconnected.")
	send(msg, h.TbAPI)
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
