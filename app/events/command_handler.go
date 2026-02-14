package events

import (
	"context"
	"fmt"
	"log"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

type BotCommandHandler struct {
	TbAPI              TbAPI
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
		h.handleStart(ctx, chatID, userID)
	case "connect":
		h.handleConnect(chatID, userID)
	case "disconnect":
		h.handleDisconnect(chatID, userID)
	case "init":
		h.handleInit(ctx, chatID, userID)
	case "sync":
		h.handleSync(ctx, chatID, userID)
	case "last":
		h.handleLast(ctx, chatID, userID)
	case "analyze":
		h.handleAnalyze(chatID, userID)
	}
}

func (h *BotCommandHandler) handleStart(ctx context.Context, chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)
	synced, _ := h.UserManager.IsSynced(userID)

	var rows [][]tbapi.InlineKeyboardButton

	if !connected {
		rows = append(rows, tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Connect Hevy Account", "connect_hevy"),
		))
	} else if !synced {
		rows = append(rows, tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Run Initial Sync", "init_sync"),
		))
	} else {
		rows = append(rows,
			tbapi.NewInlineKeyboardRow(
				tbapi.NewInlineKeyboardButtonData("Last Workout", "fetch_last"),
				tbapi.NewInlineKeyboardButtonData("Sync New", "run_sync"),
			),
			tbapi.NewInlineKeyboardRow(
				tbapi.NewInlineKeyboardButtonData("Analyze", "show_analyze"),
			),
		)
	}

	keyboard := tbapi.NewInlineKeyboardMarkup(rows...)

	text := "Welcome to GymBuddy!\n\n"
	if !connected {
		text += "Connect your Hevy account with /connect to unlock workout sync and AI analysis."
	} else if !synced {
		text += "Your Hevy account is connected! Run /init to import your workout history."
	} else {
		text += "Your Hevy account is connected and synced.\n\nCommands:\n/last — latest workout\n/sync — fetch new workouts\n/analyze — training analysis"
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

func (h *BotCommandHandler) handleInit(ctx context.Context, chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)
	if !connected {
		msg := tbapi.NewMessage(chatID, "Hevy account not connected. Use /connect first.")
		send(msg, h.TbAPI)
		return
	}

	err := h.UserManager.InitSync(ctx, userID, func(text string) {
		msg := tbapi.NewMessage(chatID, text)
		send(msg, h.TbAPI)
	})
	if err != nil {
		log.Printf("[error] init sync failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Sync failed: %v", err))
		send(msg, h.TbAPI)
	}
}

func (h *BotCommandHandler) handleSync(ctx context.Context, chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)
	if !connected {
		msg := tbapi.NewMessage(chatID, "Hevy account not connected. Use /connect first.")
		send(msg, h.TbAPI)
		return
	}

	count, err := h.UserManager.IncrementalSync(ctx, userID)
	if err != nil {
		log.Printf("[error] sync failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Sync failed: %v", err))
		send(msg, h.TbAPI)
		return
	}

	text := "Already up to date."
	if count > 0 {
		text = fmt.Sprintf("Synced %d new workout(s).", count)
	}
	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)
}

func (h *BotCommandHandler) handleLast(ctx context.Context, chatID, userID int64) {
	connected, _ := h.UserManager.IsHevyConnected(userID)
	if !connected {
		msg := tbapi.NewMessage(chatID, "Hevy account not connected. Use /connect first.")
		send(msg, h.TbAPI)
		return
	}

	w, err := h.UserManager.GetLastWorkout(ctx, userID)
	if err != nil {
		log.Printf("[error] get last workout failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Failed to fetch workout: %v", err))
		send(msg, h.TbAPI)
		return
	}
	if w == nil {
		msg := tbapi.NewMessage(chatID, "No workouts found.")
		send(msg, h.TbAPI)
		return
	}

	text := formatWorkout(w)

	// Volume comparison with previous session of same title
	previous, err := h.UserManager.GetWorkoutsByTitle(userID, w.Title)
	if err == nil && len(previous) > 1 {
		text += "\n\n" + formatVolumeComparison(w, &previous[1])
	}

	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)
}

func (h *BotCommandHandler) handleAnalyze(chatID, userID int64) {
	synced, _ := h.UserManager.IsSynced(userID)
	if !synced {
		msg := tbapi.NewMessage(chatID, "No data to analyze. Run /init first.")
		send(msg, h.TbAPI)
		return
	}

	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Volume Report", "analyze_volume"),
			tbapi.NewInlineKeyboardButtonData("Progressive Overload", "analyze_overload"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Muscle Balance", "analyze_balance"),
			tbapi.NewInlineKeyboardButtonData("Full Report", "analyze_full"),
		),
	)

	msg := tbapi.NewMessage(chatID, "Choose analysis type:")
	msg.ReplyMarkup = keyboard

	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send analyze menu: %v", err)
	}
}

func formatWorkout(w *hevy.Workout) string {
	duration := w.EndTime.Sub(w.StartTime)
	text := fmt.Sprintf("%s\n%s | %d min\n",
		w.Title,
		w.StartTime.Format("Mon, 2 Jan 2006 15:04"),
		int(duration.Minutes()),
	)

	totalSets := 0
	totalVolume := 0.0

	for _, ex := range w.Exercises {
		text += fmt.Sprintf("\n  %s", ex.Title)
		for _, s := range ex.Sets {
			totalSets++
			weight := 0.0
			reps := 0
			if s.WeightKG != nil {
				weight = *s.WeightKG
			}
			if s.Reps != nil {
				reps = *s.Reps
			}
			totalVolume += weight * float64(reps)

			if weight > 0 && reps > 0 {
				text += fmt.Sprintf("\n    %s: %.1f kg x %d", s.Type, weight, reps)
			} else if reps > 0 {
				text += fmt.Sprintf("\n    %s: %d reps", s.Type, reps)
			} else if s.DurationSeconds != nil && *s.DurationSeconds > 0 {
				text += fmt.Sprintf("\n    %s: %ds", s.Type, *s.DurationSeconds)
			}
			if s.RPE != nil && *s.RPE > 0 {
				text += fmt.Sprintf(" @RPE %.0f", *s.RPE)
			}
		}
	}

	text += fmt.Sprintf("\n\nTotal: %d sets, %.0f kg volume", totalSets, totalVolume)
	return text
}

func formatVolumeComparison(current, previous *hevy.Workout) string {
	currentVol := calcTotalVolume(current)
	previousVol := calcTotalVolume(previous)

	if previousVol == 0 {
		return ""
	}

	diff := currentVol - previousVol
	pct := (diff / previousVol) * 100

	sign := "+"
	if diff < 0 {
		sign = ""
	}

	return fmt.Sprintf("vs previous %s: %s%.0f kg (%s%.1f%%)",
		previous.StartTime.Format("2 Jan"),
		sign, diff, sign, pct)
}

func calcTotalVolume(w *hevy.Workout) float64 {
	total := 0.0
	for _, ex := range w.Exercises {
		for _, s := range ex.Sets {
			if s.WeightKG != nil && s.Reps != nil {
				total += *s.WeightKG * float64(*s.Reps)
			}
		}
	}
	return total
}
