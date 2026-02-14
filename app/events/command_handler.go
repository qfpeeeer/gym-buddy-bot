package events

import (
	"context"
	"fmt"
	"log"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

type BotCommandHandler struct {
	TbAPI                TbAPI
	UserManager          UserManager
	SetAwaitingHevyKey   func(userID, chatID int64)
	SetAwaitingOpenAIKey func(userID, chatID int64)
	SetAwaitingNotes     func(userID, chatID int64)
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
	case "setup_ai":
		h.handleSetupAI(chatID, userID)
	case "goals":
		h.handleGoals(chatID, userID)
	case "notes":
		h.handleNotes(chatID, userID)
	case "advice":
		h.handleAdvice(ctx, chatID, userID)
	case "ask":
		h.handleAsk(ctx, chatID, userID, update.Message.CommandArguments())
	case "template":
		h.handleTemplate(chatID, userID)
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
		aiConfigured, _ := h.UserManager.IsAIConfigured(userID)
		rows = append(rows,
			tbapi.NewInlineKeyboardRow(
				tbapi.NewInlineKeyboardButtonData("Last Workout", "fetch_last"),
				tbapi.NewInlineKeyboardButtonData("Sync New", "run_sync"),
			),
			tbapi.NewInlineKeyboardRow(
				tbapi.NewInlineKeyboardButtonData("Analyze", "show_analyze"),
			),
		)
		if aiConfigured {
			rows = append(rows,
				tbapi.NewInlineKeyboardRow(
					tbapi.NewInlineKeyboardButtonData("AI Advice", "get_advice"),
				),
			)
		}
	}

	keyboard := tbapi.NewInlineKeyboardMarkup(rows...)

	text := "Welcome to GymBuddy!\n\n"
	if !connected {
		text += "Connect your Hevy account with /connect to unlock workout sync and AI analysis."
	} else if !synced {
		text += "Your Hevy account is connected! Run /init to import your workout history."
	} else {
		text += "Your Hevy account is connected and synced.\n\nCommands:\n/last — latest workout\n/sync — fetch new workouts\n/analyze — training analysis\n/advice — AI recommendations\n/ask — ask your AI coach\n/template — generate workout template\n/goals — set training goals\n/notes — set training preferences\n/setup_ai — configure OpenAI"
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

	rows := [][]tbapi.InlineKeyboardButton{
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Volume Report", "analyze_volume"),
			tbapi.NewInlineKeyboardButtonData("Progressive Overload", "analyze_overload"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Muscle Balance", "analyze_balance"),
			tbapi.NewInlineKeyboardButtonData("Full Report", "analyze_full"),
		),
	}

	aiConfigured, _ := h.UserManager.IsAIConfigured(userID)
	if aiConfigured {
		rows = append(rows, tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("AI Insights", "ai_insights"),
		))
	}

	keyboard := tbapi.NewInlineKeyboardMarkup(rows...)

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

func (h *BotCommandHandler) handleSetupAI(chatID, userID int64) {
	text := "Send me your OpenAI API key.\n\nGet it from: platform.openai.com/api-keys"
	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)

	if h.SetAwaitingOpenAIKey != nil {
		h.SetAwaitingOpenAIKey(userID, chatID)
	}
}

func (h *BotCommandHandler) handleGoals(chatID, userID int64) {
	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Hypertrophy", "set_goal_hypertrophy"),
			tbapi.NewInlineKeyboardButtonData("Strength", "set_goal_strength"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Endurance", "set_goal_endurance"),
			tbapi.NewInlineKeyboardButtonData("Recomp", "set_goal_recomp"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Custom...", "set_goal_custom"),
		),
	)

	prefs, _ := h.UserManager.GetPreferences(userID)
	text := "Choose your training goal:"
	if prefs != nil && prefs.Goals != "" {
		text = fmt.Sprintf("Current goal: %s\n\nChoose a new goal:", prefs.Goals)
	}

	msg := tbapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send goals menu: %v", err)
	}
}

func (h *BotCommandHandler) handleNotes(chatID, userID int64) {
	prefs, _ := h.UserManager.GetPreferences(userID)
	text := "Send me your training notes/preferences.\n\nExamples:\n- \"Focusing on back width and tricep mass\"\n- \"Weak hamstrings, want to bring them up\"\n- \"Prefer compound movements, avoid machines\""
	if prefs != nil && prefs.Notes != "" {
		text = fmt.Sprintf("Current notes: %s\n\n%s", prefs.Notes, text)
	}

	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)

	if h.SetAwaitingNotes != nil {
		h.SetAwaitingNotes(userID, chatID)
	}
}

func (h *BotCommandHandler) handleAdvice(ctx context.Context, chatID, userID int64) {
	aiConfigured, _ := h.UserManager.IsAIConfigured(userID)
	if !aiConfigured {
		msg := tbapi.NewMessage(chatID, "OpenAI not configured. Use /setup_ai first.")
		send(msg, h.TbAPI)
		return
	}

	synced, _ := h.UserManager.IsSynced(userID)
	if !synced {
		msg := tbapi.NewMessage(chatID, "No workout data. Run /init first.")
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, "Thinking...")
	send(msg, h.TbAPI)

	result, err := h.UserManager.GetAdvice(ctx, userID)
	if err != nil {
		log.Printf("[error] get advice failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Failed: %v", err))
		send(msg, h.TbAPI)
		return
	}

	msg = tbapi.NewMessage(chatID, result)
	send(msg, h.TbAPI)
}

func (h *BotCommandHandler) handleTemplate(chatID, userID int64) {
	aiConfigured, _ := h.UserManager.IsAIConfigured(userID)
	if !aiConfigured {
		msg := tbapi.NewMessage(chatID, "OpenAI not configured. Use /setup_ai first.")
		send(msg, h.TbAPI)
		return
	}

	synced, _ := h.UserManager.IsSynced(userID)
	if !synced {
		msg := tbapi.NewMessage(chatID, "No workout data. Run /init first.")
		send(msg, h.TbAPI)
		return
	}

	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Push Day", "gen_tpl_push"),
			tbapi.NewInlineKeyboardButtonData("Pull Day", "gen_tpl_pull"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Leg Day", "gen_tpl_legs"),
			tbapi.NewInlineKeyboardButtonData("Full Body", "gen_tpl_full"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Upper Body", "gen_tpl_upper"),
			tbapi.NewInlineKeyboardButtonData("Lower Body", "gen_tpl_lower"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("AI Recommends", "gen_tpl_ai"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Custom...", "gen_tpl_custom"),
		),
	)

	msg := tbapi.NewMessage(chatID, "Choose workout type:")
	msg.ReplyMarkup = keyboard
	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send template menu: %v", err)
	}
}

func (h *BotCommandHandler) handleAsk(ctx context.Context, chatID, userID int64, question string) {
	if question == "" {
		msg := tbapi.NewMessage(chatID, "Usage: /ask <your question>\n\nExample: /ask Should I deload this week?")
		send(msg, h.TbAPI)
		return
	}

	aiConfigured, _ := h.UserManager.IsAIConfigured(userID)
	if !aiConfigured {
		msg := tbapi.NewMessage(chatID, "OpenAI not configured. Use /setup_ai first.")
		send(msg, h.TbAPI)
		return
	}

	synced, _ := h.UserManager.IsSynced(userID)
	if !synced {
		msg := tbapi.NewMessage(chatID, "No workout data. Run /init first.")
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, "Thinking...")
	send(msg, h.TbAPI)

	result, err := h.UserManager.AskQuestion(ctx, userID, question)
	if err != nil {
		log.Printf("[error] ask question failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Failed: %v", err))
		send(msg, h.TbAPI)
		return
	}

	msg = tbapi.NewMessage(chatID, result)
	send(msg, h.TbAPI)
}
