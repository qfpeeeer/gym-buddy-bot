# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GymBuddy is a Telegram bot (Go) that acts as a fitness companion — it integrates with the **Strong** workout app, lets users import workouts (via shared text or CSV export), provides an exercise database with 800+ exercises, and generates random workout plans.

## Build & Run Commands

```bash
# Build
go build -o gym-buddy-bot ./app

# Run
go run ./app/main.go

# Run all tests
go test ./...

# Run a single package's tests
go test ./app/strong/...

# Run a single test
go test -run TestParseSingleWorkout ./app/strong/...

# Vet
go vet ./...
```

## Environment Variables

Set in `deployments/.env` (see `deployments/example.env`):

- `TELEGRAM_TOKEN` — Telegram bot token (required)
- `DATA_FILE_PATH` — SQLite database file path, e.g. `data.db` (required)
- `REVISION` — environment label shown at startup (optional)

## Architecture

**Entry point:** `app/main.go` — loads env vars, opens SQLite DB, initializes `UserManager` and `ExerciseManager`, starts the Telegram listener.

### Event Handling Layer (`app/events/`)

`TelegramListener` polls Telegram for updates and dispatches to three handlers based on update type:

- **CommandHandler** — bot commands (`/start`, `/history`)
- **MessageHandler** — processes document uploads (Strong CSV) and shared workout text
- **CallbackQueryHandler** — inline keyboard button presses (`get_exercises`, `exercise_info_{id}`, `remove_exercise_{id}`, `replace_exercise_{id}`, `back_to_exercises`)

All handlers receive `UserManager` and `ExerciseManager` as dependencies. The `events.go` file defines shared interfaces and utility functions.

### Business Logic

- **`app/user/user.go`** — `UserManager` is the main facade that coordinates user operations, exercise management, and workout storage. Handlers call into this rather than storage directly.
- **`app/exercises/exercises.go`** — `ExerciseManager` loads `exercises.json` at startup and provides exercise lookup/random selection.

### Storage Layer (`app/storage/`)

SQLite via `sqlx` + `modernc.org/sqlite` (pure Go, no CGo). Schema defined in `schema.sql`.

- `storage.go` — DB connection factory
- `user.go` — user CRUD
- `exercise.go` — user-exercise associations
- `workout.go` — workout and workout_set persistence

Key tables: `users`, `user_exercises`, `workouts`, `workout_sets`. See `schema.sql` for full schema including indexes and foreign keys.

### Strong App Parsers (`app/strong/`)

- **`text_parser.go`** — parses workout text shared from Strong app. Detects format by checking for day-of-week on line 2. Supports European decimal format (e.g. `17,5 kg`) and bodyweight exercises.
- **`csv_parser.go`** — parses Strong CSV exports, groups sets by workout date/name, handles duplicate detection.
- **`model.go`** — shared data structures (`Workout`, `WorkoutSet`).

Both parsers have dedicated test files with good coverage.

## Key Patterns

- The bot uses `go-telegram-bot-api/v5` — messages are sent via `tgbotapi.NewMessage()`, inline keyboards via `tgbotapi.NewInlineKeyboardMarkup()`, and callbacks answered with `tgbotapi.NewCallback()`.
- Callback data uses underscore-delimited prefixes for routing (e.g. `exercise_info_`, `remove_exercise_`).
- The module path is `github.com/qfpeeeer/gym-buddy-bot`.
