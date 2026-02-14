package events

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/qfpeeeer/gym-buddy-bot/app/llm"
)

type pendingTemplate struct {
	template    *llm.TemplateResponse
	workoutType string
}

type BotCallbackQueryHandler struct {
	TbAPI                    TbAPI
	UserManager              UserManager
	SetAwaitingHevyKey       func(userID, chatID int64)
	SetAwaitingOpenAIKey     func(userID, chatID int64)
	SetAwaitingNotes         func(userID, chatID int64)
	SetAwaitingGoalText      func(userID, chatID int64)
	SetAwaitingCustomTplType func(userID, chatID int64)
	SetAwaitingRegenNotes    func(userID, chatID int64)

	mu               sync.Mutex
	pendingTemplates map[int64]*pendingTemplate
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
		h.handleShowAnalyze(ctx, query)
	case data == "ai_insights":
		h.handleAIInsights(ctx, query)
	case data == "get_advice":
		h.handleGetAdvice(ctx, query)
	case strings.HasPrefix(data, "analyze_"):
		h.handleAnalyze(query)
	case strings.HasPrefix(data, "set_goal_"):
		h.handleSetGoal(query)
	case strings.HasPrefix(data, "set_model_"):
		h.handleSetModel(query)
	case strings.HasPrefix(data, "gen_tpl_"):
		h.handleGenTemplate(ctx, query)
	case data == "push_routine":
		h.handlePushRoutine(ctx, query)
	case data == "regen_template":
		h.handleRegenTemplate(ctx, query)
	case data == "regen_with_notes":
		h.handleRegenWithNotes(query)
	case data == "back_to_tpl_menu":
		h.handleBackToTemplateMenu(query)
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

func (h *BotCallbackQueryHandler) handleShowAnalyze(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "")
	h.TbAPI.Request(callback)

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

func (h *BotCallbackQueryHandler) handleAIInsights(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "Thinking...")
	h.TbAPI.Request(callback)

	// Run full analysis first
	analysisText, err := h.UserManager.RunAnalysis(userID, "")
	if err != nil {
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Analysis failed: %v", err))
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, "Analyzing with AI...")
	send(msg, h.TbAPI)

	result, err := h.UserManager.GetAnalysisInsights(ctx, userID, analysisText)
	if err != nil {
		log.Printf("[error] AI insights failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Failed: %v", err))
		send(msg, h.TbAPI)
		return
	}

	msg = tbapi.NewMessage(chatID, result)
	send(msg, h.TbAPI)
}

func (h *BotCallbackQueryHandler) handleGetAdvice(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "Thinking...")
	h.TbAPI.Request(callback)

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

func (h *BotCallbackQueryHandler) handleSetGoal(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	goal := strings.TrimPrefix(query.Data, "set_goal_")

	callback := tbapi.NewCallback(query.ID, "")
	h.TbAPI.Request(callback)

	if goal == "custom" {
		msg := tbapi.NewMessage(chatID, "Send me your custom training goal:")
		send(msg, h.TbAPI)
		if h.SetAwaitingGoalText != nil {
			h.SetAwaitingGoalText(userID, chatID)
		}
		return
	}

	// Capitalize
	goalText := strings.ToUpper(goal[:1]) + goal[1:]

	if err := h.UserManager.SetGoals(userID, goalText); err != nil {
		log.Printf("[error] failed to set goal: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to save goal.")
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, fmt.Sprintf("Goal set: %s", goalText))
	send(msg, h.TbAPI)
}

func (h *BotCallbackQueryHandler) handleSetModel(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	model := strings.TrimPrefix(query.Data, "set_model_")

	callback := tbapi.NewCallback(query.ID, "")
	h.TbAPI.Request(callback)

	if err := h.UserManager.SetAIModel(userID, model); err != nil {
		log.Printf("[error] failed to set model: %v", err)
		msg := tbapi.NewMessage(chatID, "Failed to save model.")
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, fmt.Sprintf("Model set: %s", model))
	send(msg, h.TbAPI)
}

// --- Template generation callbacks ---

var templateTypeNames = map[string]string{
	"push":   "Push Day",
	"pull":   "Pull Day",
	"legs":   "Leg Day",
	"full":   "Full Body",
	"upper":  "Upper Body",
	"lower":  "Lower Body",
	"ai":     "workout based on what this client needs most right now (analyze their volume gaps, stalling exercises, and muscle balance to determine the optimal workout type)",
	"custom": "Custom",
}

func (h *BotCallbackQueryHandler) setPending(userID int64, p *pendingTemplate) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pendingTemplates == nil {
		h.pendingTemplates = make(map[int64]*pendingTemplate)
	}
	h.pendingTemplates[userID] = p
}

func (h *BotCallbackQueryHandler) getPending(userID int64) *pendingTemplate {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pendingTemplates[userID]
}

func (h *BotCallbackQueryHandler) clearPending(userID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.pendingTemplates, userID)
}

func (h *BotCallbackQueryHandler) handleGenTemplate(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	tplType := strings.TrimPrefix(query.Data, "gen_tpl_")

	callback := tbapi.NewCallback(query.ID, "Generating...")
	h.TbAPI.Request(callback)

	if tplType == "custom" {
		msg := tbapi.NewMessage(chatID, "Describe the workout you want:\n\nExample: \"Back and biceps with emphasis on pullups\" or \"Arms isolation day\"")
		send(msg, h.TbAPI)
		if h.SetAwaitingCustomTplType != nil {
			h.SetAwaitingCustomTplType(userID, chatID)
		}
		return
	}

	typeName := templateTypeNames[tplType]
	if typeName == "" {
		typeName = tplType
	}

	h.GenerateAndPreview(ctx, userID, chatID, tplType, typeName, "")
}

// GenerateAndPreview generates a template via LLM and shows a preview with action buttons.
func (h *BotCallbackQueryHandler) GenerateAndPreview(ctx context.Context, userID, chatID int64, tplType, typeName, extraNotes string) {
	msg := tbapi.NewMessage(chatID, fmt.Sprintf("Generating %s template...", typeName))
	send(msg, h.TbAPI)

	tmpl, err := h.UserManager.GenerateTemplate(ctx, userID, typeName, extraNotes)
	if err != nil {
		log.Printf("[error] template generation failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Failed: %v", err))
		send(msg, h.TbAPI)
		return
	}

	h.setPending(userID, &pendingTemplate{template: tmpl, workoutType: tplType})

	preview := llm.FormatTemplatePreview(tmpl)

	keyboard := tbapi.NewInlineKeyboardMarkup(
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Push to Hevy", "push_routine"),
			tbapi.NewInlineKeyboardButtonData("Regenerate", "regen_template"),
		),
		tbapi.NewInlineKeyboardRow(
			tbapi.NewInlineKeyboardButtonData("Regenerate with notes", "regen_with_notes"),
			tbapi.NewInlineKeyboardButtonData("Back to menu", "back_to_tpl_menu"),
		),
	)

	msg = tbapi.NewMessage(chatID, preview)
	msg.ReplyMarkup = keyboard
	if _, err := h.TbAPI.Send(msg); err != nil {
		log.Printf("[error] failed to send template preview: %v", err)
		// Fallback without keyboard
		send(tbapi.NewMessage(chatID, preview), h.TbAPI)
	}
}

func (h *BotCallbackQueryHandler) handlePushRoutine(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "Pushing to Hevy...")
	h.TbAPI.Request(callback)

	pending := h.getPending(userID)
	if pending == nil {
		msg := tbapi.NewMessage(chatID, "No pending template. Use /template to generate one.")
		send(msg, h.TbAPI)
		return
	}

	err := h.UserManager.PushRoutineToHevy(ctx, userID, pending.template)
	if err != nil {
		log.Printf("[error] push routine failed for user %d: %v", userID, err)
		msg := tbapi.NewMessage(chatID, fmt.Sprintf("Failed: %v", err))
		send(msg, h.TbAPI)
		return
	}

	h.clearPending(userID)

	msg := tbapi.NewMessage(chatID, fmt.Sprintf("Routine \"%s\" created in Hevy!", pending.template.Title))
	send(msg, h.TbAPI)
}

func (h *BotCallbackQueryHandler) handleRegenTemplate(ctx context.Context, query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "Regenerating...")
	h.TbAPI.Request(callback)

	pending := h.getPending(userID)
	if pending == nil {
		msg := tbapi.NewMessage(chatID, "No pending template. Use /template to generate one.")
		send(msg, h.TbAPI)
		return
	}

	typeName := templateTypeNames[pending.workoutType]
	if typeName == "" {
		typeName = pending.workoutType
	}

	h.GenerateAndPreview(ctx, userID, chatID, pending.workoutType, typeName, "")
}

func (h *BotCallbackQueryHandler) handleRegenWithNotes(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "")
	h.TbAPI.Request(callback)

	pending := h.getPending(userID)
	if pending == nil {
		msg := tbapi.NewMessage(chatID, "No pending template. Use /template to generate one.")
		send(msg, h.TbAPI)
		return
	}

	msg := tbapi.NewMessage(chatID, "What would you like to change?\n\nExample: \"More back width exercises, drop the curls\" or \"Add a warmup set for each exercise\"")
	send(msg, h.TbAPI)

	if h.SetAwaitingRegenNotes != nil {
		h.SetAwaitingRegenNotes(userID, chatID)
	}
}

func (h *BotCallbackQueryHandler) handleBackToTemplateMenu(query *tbapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID

	callback := tbapi.NewCallback(query.ID, "")
	h.TbAPI.Request(callback)

	h.clearPending(userID)

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
