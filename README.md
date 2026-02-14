# GymBuddy Bot

A Telegram bot that integrates with [Hevy](https://hevy.com) to act as an AI-powered training coach. Hevy handles workout logging in the gym; the bot provides analysis, recommendations, and routine generation.

## Features

- **Hevy Integration** — connect your Hevy account via API key, sync workouts automatically
- **Workout Sync** — full initial sync (`/init`) and incremental updates (`/sync`)
- **Latest Workout** — view your most recent session with volume comparison (`/last`)
- **Training Analysis** — algorithmic analysis of your training data (`/analyze`):
  - Volume report — weekly sets & tonnage per muscle group
  - Progressive overload — estimated 1RM trends (Epley formula)
  - Muscle balance — push/pull/legs ratios with imbalance warnings
  - Training frequency — workouts/week, session duration, rest days

## Setup

### Prerequisites

- Go 1.21+
- Telegram bot token (from [@BotFather](https://t.me/BotFather))
- Hevy Pro subscription (for API access)

### Configuration

Copy `deployments/example.env` to `deployments/.env` and fill in:

```env
TELEGRAM_TOKEN=your-telegram-bot-token
DATA_FILE_PATH=data.db
HEVY_API_KEY=your-hevy-api-key
```

### Build & Run

```bash
go build -o gym-buddy-bot ./app
./gym-buddy-bot
```

## Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Show main menu (adapts to connection/sync state) |
| `/connect` | Connect your Hevy account |
| `/disconnect` | Disconnect Hevy account |
| `/init` | Full sync — import all workouts + exercise templates |
| `/sync` | Incremental sync — fetch new workouts |
| `/last` | Show latest workout with volume comparison |
| `/analyze` | Training analysis (volume, overload, balance, full report) |

## Architecture

```
app/
  main.go              — entry point, wiring
  hevy/                — Hevy REST API client (workouts, exercises, routines)
  storage/             — SQLite storage (users, workouts, exercise cache, sync state)
  user/                — business logic facade (sync orchestration, analysis)
  analysis/            — training analysis engine (volume, overload, balance, frequency)
  events/              — Telegram event handlers (commands, messages, callbacks)
scripts/
  import_to_hevy.go    — one-time Strong CSV to Hevy import script
```

## Roadmap

See [PLAN.md](PLAN.md) for the full implementation plan.

- [x] Phase 1: Hevy API client + user connection
- [x] Phase 2: Workout sync + exercise template cache
- [x] Phase 3: Analysis engine
- [ ] Phase 4: LLM integration (Claude/OpenAI) for smart recommendations
- [ ] Phase 5: AI-powered routine generation + push to Hevy
