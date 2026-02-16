# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GymBuddy is a Telegram bot (Go) that acts as an AI-powered fitness coach. It integrates with the **Hevy** workout tracker via its REST API, syncs workout history to a local SQLite database, provides training analysis (volume, progressive overload, muscle balance, frequency), offers AI-powered coaching via OpenAI, and generates workout templates pushed directly to Hevy.

## Build & Run Commands

```bash
# Build
go build -o gym-buddy-bot ./app

# Run
go run ./app/main.go

# Vet
go vet ./...
```

## Environment Variables

Set in `deployments/.env` (see `deployments/example.env`):

- `TELEGRAM_TOKEN` — Telegram bot token (required)
- `DATA_FILE_PATH` — SQLite database file path, e.g. `data.db` (required)
- `REVISION` — environment label shown at startup (optional)

AI settings (OpenAI key + model) are per-user, stored in the `ai_settings` table, configured via `/setup_ai` in Telegram.

## Architecture

**Entry point:** `app/main.go` — loads env vars, opens SQLite DB, initializes all storage layers (user, workout, exercise cache, sync, AI settings, preferences) and `UserManager`, starts the Telegram listener.

### Event Handling Layer (`app/events/`)

`TelegramListener` polls Telegram for updates and dispatches to three handlers based on update type:

- **CommandHandler** — bot commands (`/start`, `/connect`, `/disconnect`, `/init`, `/sync`, `/last`, `/analyze`, `/setup_ai`, `/goals`, `/notes`, `/advice`, `/ask`, `/template`)
- **MessageHandler** — processes incoming text messages. Manages awaiting states: `awaitingHevyKey`, `awaitingAIKey`, `awaitingNotes`, `awaitingGoalText`, `awaitingCustomTpl`, `awaitingRegenNotes`
- **CallbackQueryHandler** — inline keyboard button presses. Manages `pendingTemplates` map for template preview/push/regenerate flow.

All handlers receive `UserManager` as their dependency. The `events.go` file defines the `UserManager` interface and shared utility functions.

Callback data prefixes:
- Connection: `connect_hevy`
- Sync: `init_sync`, `run_sync`
- Workouts: `fetch_last`
- Analysis: `show_analyze`, `analyze_volume`, `analyze_overload`, `analyze_balance`, `analyze_full`, `ai_insights`
- AI: `get_advice`, `set_goal_*`, `set_model_*`
- Templates: `gen_tpl_push`, `gen_tpl_pull`, `gen_tpl_legs`, `gen_tpl_full`, `gen_tpl_upper`, `gen_tpl_lower`, `gen_tpl_ai`, `gen_tpl_custom`, `push_routine`, `regen_template`, `regen_with_notes`, `back_to_tpl_menu`

### Business Logic (`app/user/`)

`UserManager` is the main facade. It coordinates:
- User operations and Hevy API calls
- Workout sync (full and incremental)
- Training analysis
- AI advice and question answering (`GetAdvice`, `AskQuestion`, `GetAnalysisInsights`)
- Template generation and Hevy push (`GenerateTemplate`, `PushRoutineToHevy`)

Defines storage interfaces (`Storage`, `WorkoutStorage`, `ExerciseCacheStorage`, `SyncStorage`, `AISettingsStorage`, `PreferencesStorage`) to decouple from concrete implementations.

### Hevy API Client (`app/hevy/`)

- **`client.go`** — HTTP client for the Hevy REST API (`https://api.hevyapp.com/v1`). Handles pagination, rate limiting, auth via `api-key` header. Provides `get()`, `post()`, `put()` helpers.
- **`models.go`** — Go structs matching Hevy API responses: `Workout`, `Exercise`, `Set`, `ExerciseTemplate`, `Routine`, `CreateRoutineRequest`, `CreateRoutineResponse`.
- **`workouts.go`** — workout fetch endpoints (all, single, paginated).
- **`exercises.go`** — exercise template fetch endpoints.
- **`routines.go`** — routine CRUD: `GetRoutines`, `GetAllRoutines`, `CreateRoutine`, `UpdateRoutine`, `GetRoutineFolders`, `CreateRoutineFolder`.

### LLM Integration (`app/llm/`)

OpenAI chat completions via standard library `net/http` (no SDK dependency):

- **`client.go`** — `Client` with `Chat(ctx, systemPrompt, userMessage)`. Default model: `gpt-4.1-mini`. 60s timeout.
- **`prompts.go`** — System prompts: `CoachSystemPrompt` (elite strength coach persona), `AnalysisInsightsPrompt` (analysis interpretation), `TemplateGenerationPrompt` (structured JSON output with exercise validation rules).
- **`context.go`** — `BuildContext()` assembles compact training context from local data (profile, preferences, volume, overload, balance, recent workouts). `BuildContextWithExercises()` adds available exercises section with IDs, muscle groups, equipment, and last weights used.
- **`template.go`** — `ParseTemplateJSON()` extracts JSON from LLM response. `ValidateAndFixTemplate()` validates exercise IDs against cache. `FormatTemplatePreview()` for Telegram display. `ToCreateRoutineRequest()` converts to Hevy API format.

### Analysis Engine (`app/analysis/`)

Pure Go analysis with no external dependencies:

- **`models.go`** — Result types: `AnalysisResult`, `VolumeReport`, `OverloadReport`, `BalanceReport`, `FrequencyReport`
- **`volume.go`** — Weekly sets per muscle group (primary: 1.0, secondary: 0.5 credit, excludes warmup), tonnage, status thresholds (<10 low, 10-20 optimal, >20 high)
- **`progressive.go`** — Estimated 1RM tracking via Epley formula (`weight * (1 + reps/30)`), trend detection by comparing last 3 vs previous 3 sessions (>2.5% change threshold)
- **`balance.go`** — Push/Pull/Legs classification, ratio analysis with imbalance flags
- **`frequency.go`** — Workouts per week, session duration, per-muscle frequency, rest days
- **`engine.go`** — `Analyze()` coordinator + `Format*()` functions for Telegram-friendly text output

### Storage Layer (`app/storage/`)

SQLite via `sqlx` + `modernc.org/sqlite` (pure Go, no CGo). Schema defined in `schema.sql`.

- `storage.go` — DB connection factory
- `user.go` — user CRUD (telegram_id, hevy_api_key)
- `workout.go` — workout, exercise, and set persistence (synced from Hevy)
- `exercise_cache.go` — cached Hevy exercise templates for muscle group lookups
- `sync.go` — per-user sync state tracking (last sync time, workout count)
- `ai_settings.go` — per-user OpenAI key + model selection (`ai_settings` table)
- `preferences.go` — user training goals + notes (`user_preferences` table)

Key tables: `users`, `workouts`, `workout_exercises`, `workout_sets`, `exercise_templates`, `sync_state`, `ai_settings`, `user_preferences`.

## Key Patterns

- The bot uses `go-telegram-bot-api/v5` — messages are sent via `tgbotapi.NewMessage()`, inline keyboards via `tgbotapi.NewInlineKeyboardMarkup()`, and callbacks answered with `tgbotapi.NewCallback()`.
- Callback data uses underscore-delimited prefixes for routing (e.g. `connect_hevy`, `analyze_volume`, `gen_tpl_push`).
- The `/start` command shows an adaptive menu with states: not connected, connected but not synced, synced, synced + AI configured.
- The module path is `github.com/qfpeeeer/gym-buddy-bot`.
- Hevy API auth is per-user — each user provides their own API key via `/connect`. Keys are stored in the `users` table.
- OpenAI auth is per-user — each user provides their own API key via `/setup_ai`. Keys are stored in the `ai_settings` table. API key messages are deleted immediately after capture for security.
- Analysis runs entirely on locally synced data (no API calls needed after sync).
- AI features gracefully degrade — if no OpenAI key configured, algorithmic-only results are shown.
- Template generation flow: generate → preview → push to Hevy / regenerate / regenerate with notes. Templates are ephemeral (not persisted locally).
- LLM integration uses only `net/http` — no OpenAI SDK dependency.

## Hevy API Reference

- **Swagger docs:** https://api.hevyapp.com/docs/
- **Base URL:** `https://api.hevyapp.com/v1`
- **Auth:** `api-key` header (each user gets their key from https://hevy.com/settings?developer)
- **Requires:** Hevy Pro subscription for API access

### Hevy Data Model Enums

**Exercise types:** `weight_reps`, `reps_only`, `bodyweight_reps`, `bodyweight_assisted_reps`, `duration`, `weight_duration`, `distance_duration`, `short_distance_weight`

**Muscle groups:** `abdominals`, `shoulders`, `biceps`, `triceps`, `forearms`, `quadriceps`, `hamstrings`, `calves`, `glutes`, `abductors`, `adductors`, `lats`, `upper_back`, `traps`, `lower_back`, `chest`, `cardio`, `neck`, `full_body`, `other`

**Set types:** `normal`, `warmup`, `dropset`, `failure`
