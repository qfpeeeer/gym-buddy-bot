# GymBuddy Bot

A Telegram bot that integrates with [Hevy](https://hevy.com) to act as an AI-powered training coach. Hevy handles workout logging in the gym; the bot provides analysis, smart recommendations, and AI-generated workout templates pushed directly to Hevy.

## Features

- **Hevy Integration** — connect your Hevy account via API key, sync workouts automatically
- **Workout Sync** — full initial sync (`/init`) and incremental updates (`/sync`)
- **Latest Workout** — view your most recent session with volume comparison (`/last`)
- **Training Analysis** — algorithmic analysis of your training data (`/analyze`):
  - Volume report — weekly sets & tonnage per muscle group
  - Progressive overload — estimated 1RM trends (Epley formula)
  - Muscle balance — push/pull/legs ratios with imbalance warnings
  - Training frequency — workouts/week, session duration, rest days
- **AI Coach** — OpenAI-powered coaching using your real training data:
  - `/advice` — personalized recommendations based on volume, overload, and balance analysis
  - `/ask <question>` — ask your AI coach anything about your training
  - AI Insights — natural language interpretation of analysis reports
- **Workout Template Generation** — AI creates workout routines and pushes them to Hevy:
  - `/template` — choose workout type (Push/Pull/Legs/Full/Upper/Lower/AI Recommends/Custom)
  - Preview generated template before pushing
  - Regenerate with custom notes (e.g. "more back width, drop curls")
  - Routine appears directly in Hevy app

## Setup

### Prerequisites

- Go 1.21+
- Telegram bot token (from [@BotFather](https://t.me/BotFather))
- Hevy Pro subscription (for API access)
- OpenAI API key (optional, for AI coach features)

### Configuration

Copy `deployments/example.env` to `deployments/.env` and fill in:

```env
TELEGRAM_TOKEN=your-telegram-bot-token
DATA_FILE_PATH=data.db
```

AI features are configured per-user via the `/setup_ai` command in Telegram (each user provides their own OpenAI API key).

### Build & Run

```bash
go build -o gym-buddy-bot ./app
./gym-buddy-bot
```

## Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Show main menu (adapts to connection/sync/AI state) |
| `/connect` | Connect your Hevy account |
| `/disconnect` | Disconnect Hevy account |
| `/init` | Full sync — import all workouts + exercise templates |
| `/sync` | Incremental sync — fetch new workouts |
| `/last` | Show latest workout with volume comparison |
| `/analyze` | Training analysis (volume, overload, balance, full report, AI insights) |
| `/setup_ai` | Configure OpenAI API key + model selection |
| `/goals` | Set training goals (Hypertrophy/Strength/Endurance/Recomp/Custom) |
| `/notes` | Set training notes and preferences |
| `/advice` | Get AI coaching recommendations |
| `/ask <question>` | Ask your AI coach a question |
| `/template` | Generate a workout template with AI |

## Architecture

```
app/
  main.go              — entry point, wiring
  hevy/                — Hevy REST API client
    client.go          — HTTP client with pagination, rate limiting
    models.go          — API response structs
    workouts.go        — workout endpoints
    exercises.go       — exercise template endpoints
    routines.go        — routine CRUD endpoints (create, update, fetch)
  storage/             — SQLite storage layer
    storage.go         — DB connection factory
    user.go            — user CRUD
    workout.go         — workout/exercise/set persistence
    exercise_cache.go  — Hevy exercise template cache
    sync.go            — per-user sync state
    ai_settings.go     — per-user OpenAI key + model settings
    preferences.go     — user training goals + notes
  analysis/            — training analysis engine (pure Go)
    engine.go          — coordinator + Telegram formatters
    models.go          — result types
    volume.go          — weekly sets & tonnage per muscle group
    progressive.go     — estimated 1RM trends (Epley formula)
    balance.go         — push/pull/legs ratio analysis
    frequency.go       — workout frequency & rest days
  llm/                 — AI coach (OpenAI integration, no SDK)
    client.go          — OpenAI chat completions via net/http
    prompts.go         — system prompts (coach persona, analysis, template generation)
    context.go         — training context builder for LLM (profile, volume, overload, recent workouts)
    template.go        — template JSON parsing, validation, preview, Hevy conversion
  user/                — business logic facade
    user.go            — UserManager: sync, analysis, AI advice, template generation
  events/              — Telegram event handlers
    listener.go        — update polling + dispatch
    events.go          — UserManager interface + shared utilities
    command_handler.go — bot command handlers
    message_handler.go — text message handlers (API key input, custom goals, regen notes)
    callback_query_handler.go — inline keyboard button handlers
```
