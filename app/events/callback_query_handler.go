package events

import (
	"context"
	"fmt"
	"log"
	"strings"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotCallbackQueryHandler struct {
	TbAPI              TbAPI
	UserManager        UserManager
	SetAwaitingHevyKey func(userID, chatID int64)
}

func (h *BotCallbackQueryHandler) HandleCallbackQuery(ctx context.Context, update tbapi.Update) {
	query := update.CallbackQuery
	userID := query.From.ID
	data := query.Data

	log.Printf("[info] received callback query from user %d: %s", userID, data)

	switch {
	case data == "connect_hevy":
		h.handleConnectHevy(query)
	case data == "init_sync":
		h.handleInitSync(ctx, query)
	case data == "run_sync":
		h.handleRunSync(ctx, query)
	case data == "fetch_last":
		h.handleFetchLast(ctx, query)
	case data == "show_analyze":
		h.handleShowAnalyze(query)
	case strings.HasPrefix(data, "analyze_"):
		h.handleAnalyze(query)
	}
}

func (h *BotCallbackQueryHandler) handleConnectHevy(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "")
	h.TbAPI.Request(callback)

	text := "Send me your Hevy API key.\n\nGet it from: hevy.com/settings > Developer section.\nRequires Hevy Pro subscription."
	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)

	if h.SetAwaitingHevyKey != nil {
		h.SetAwaitingHevyKey(userID, chatID)
	}
}

func (h *BotCallbackQueryHandler) handleInitSync(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "Starting sync...")
	h.TbAPI.Request(callback)

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

func (h *BotCallbackQueryHandler) handleRunSync(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "Syncing...")
	h.TbAPI.Request(callback)

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

func (h *BotCallbackQueryHandler) handleFetchLast(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "Fetching...")
	h.TbAPI.Request(callback)

	w, err := h.UserManager.GetLastWorkout(ctx, userID)
	if err != nil {
		log.Printf("[error] get last workout failed: %v", err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Failed: %v", err))
		send(msg, h.TbAPI)
		return
	}
	if w == nil {
		msg := tbapi.NewMessage(chatID, "No workouts found.")
		send(msg, h.TbAPI)
		return
	}

	text := formatWorkout(w)

	previous, err := h.UserManager.GetWorkoutsByTitle(userID, w.Title)
	if err == nil && len(previous) > 1 {
		text += "\n\n" + formatVolumeComparison(w, &previous[1])
	}

	msg := tbapi.NewMessage(chatID, text)
	send(msg, h.TbAPI)
}

func (h *BotCallbackQueryHandler) handleShowAnalyze(query *tbapi.CallbackQuery) {
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "")
	h.TbAPI.Request(callback)

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

func (h *BotCallbackQueryHandler) handleAnalyze(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	reportType := strings.TrimPrefix(query.Data, "analyze_")

	callback := tbapi.NewCallback(query.ID, "Analyzing...")
	h.TbAPI.Request(callback)

	result, err := h.UserManager.RunAnalysis(userID, reportType)
	if err != nil {
		log.Printf("[error] analysis failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Analysis failed: %v", err))
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, result)
	send(msg, h.TbAPI)
}
