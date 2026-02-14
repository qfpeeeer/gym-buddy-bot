# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GymBuddy is a Telegram bot (Go) that acts as an AI-powered fitness coach. It integrates with the **Hevy** workout tracker via its REST API, syncs workout history to a local SQLite database, and provides training analysis (volume, progressive overload, muscle balance, frequency).

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

## Architecture

**Entry point:** `app/main.go` — loads env vars, opens SQLite DB, initializes all storage layers and `UserManager`, starts the Telegram listener.

### Event Handling Layer (`app/events/`)

`TelegramListener` polls Telegram for updates and dispatches to three handlers based on update type:

- **CommandHandler** — bot commands (`/start`, `/connect`, `/disconnect`, `/init`, `/sync`, `/last`, `/analyze`)
- **MessageHandler** — processes incoming text messages (currently handles Hevy API key input during connection flow)
- **CallbackQueryHandler** — inline keyboard button presses (`connect_hevy`, `init_sync`, `run_sync`, `fetch_last`, `show_analyze`, `analyze_volume`, `analyze_overload`, `analyze_balance`, `analyze_full`)

All handlers receive `UserManager` as their dependency. The `events.go` file defines the `UserManager` interface and shared utility functions.

### Business Logic

- **`app/user/user.go`** — `UserManager` is the main facade. It coordinates user operations, Hevy API calls, workout sync, and analysis. Defines storage interfaces (`Storage`, `WorkoutStorage`, `ExerciseCacheStorage`, `SyncStorage`) to decouple from concrete implementations.

### Hevy API Client (`app/hevy/`)

- **`client.go`** — HTTP client for the Hevy REST API (`https://api.hevyapp.com/v1`). Handles pagination, rate limiting, auth via `api-key` header.
- **`models.go`** — Go structs matching Hevy API responses: `Workout`, `Exercise`, `Set`, `ExerciseTemplate`, `Routine`.

Key API interactions:
- Fetching all workouts (paginated) during `/init` sync
- Fetching page 1 for incremental `/sync`
- Fetching exercise templates for muscle group data
- Individual workout fetch for `/last`

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
- `user.go` — user CRUD (telegram_id, username, hevy_api_key, state)
- `workout.go` — workout, exercise, and set persistence (synced from Hevy)
- `exercise_cache.go` — cached Hevy exercise templates for muscle group lookups
- `sync.go` — per-user sync state tracking (last sync time, workout count)

Key tables: `users`, `workouts`, `workout_exercises`, `workout_sets`, `exercise_templates`, `sync_state`. See `schema.sql` for full schema.

## Key Patterns

- The bot uses `go-telegram-bot-api/v5` — messages are sent via `tgbotapi.NewMessage()`, inline keyboards via `tgbotapi.NewInlineKeyboardMarkup()`, and callbacks answered with `tgbotapi.NewCallback()`.
- Callback data uses underscore-delimited prefixes for routing (e.g. `connect_hevy`, `analyze_volume`).
- The `/start` command shows an adaptive menu with 3 states: not connected, connected but not synced, fully synced.
- The module path is `github.com/qfpeeeer/gym-buddy-bot`.
- Hevy API auth is per-user — each user provides their own API key via `/connect`. Keys are stored in the `users` table.
- Analysis runs entirely on locally synced data (no API calls needed after sync).

## Hevy API Reference

- **Swagger docs:** https://api.hevyapp.com/docs/
- **Base URL:** `https://api.hevyapp.com/v1`
- **Auth:** `api-key` header (each user gets their key from https://hevy.com/settings?developer)
- **Requires:** Hevy Pro subscription for API access

### Hevy Data Model Enums

**Exercise types:** `weight_reps`, `reps_only`, `bodyweight_reps`, `bodyweight_assisted_reps`, `duration`, `weight_duration`, `distance_duration`, `short_distance_weight`

**Muscle groups:** `abdominals`, `shoulders`, `biceps`, `triceps`, `forearms`, `quadriceps`, `hamstrings`, `calves`, `glutes`, `abductors`, `adductors`, `lats`, `upper_back`, `traps`, `lower_back`, `chest`, `cardio`, `neck`, `full_body`, `other`

**Set types:** `normal`, `warmup`, `dropset`, `failure`
