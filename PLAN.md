# GymBuddy Bot: Hevy Integration + AI Coach

## Context

GymBuddy is a Telegram bot that acts as an AI-powered training coach. It integrates with the Hevy workout tracking app via its REST API — Hevy handles gym logging, the bot provides analysis, recommendations, and template creation.

---

## Implementation Status

| Phase | What | Status |
|-------|------|--------|
| 1 | Hevy API client + `/connect` + `/disconnect` | DONE |
| Pre | Import 52 Strong workouts + 13 routines to Hevy via API | DONE |
| Cleanup | Remove Strong parsers, exercise DB, legacy features | DONE |
| 2 | Workout sync (`/init`, `/sync`, `/last`) + exercise template cache | DONE |
| 3 | Analysis engine (volume, overload, balance, frequency) + `/analyze` | DONE |
| 4 | LLM integration (Claude/OpenAI) + `/advice` | TODO |
| 5 | Template generation + push to Hevy + `/template` | TODO |

---

## Phase 1: Hevy API Client + User Connection [DONE]

- Package `app/hevy/` — HTTP client with retry, pagination, rate limiting
- Models for Workout, ExerciseTemplate, Routine, and all CRUD request/response types
- Endpoints: workouts, exercise_templates, routines, routine_folders
- Per-user API key storage in SQLite (`hevy_api_key` column on `users` table)
- `/connect` command — prompts for API key, validates via Hevy API, stores securely (deletes user's message)
- `/disconnect` command — clears stored key
- Adaptive `/start` menu based on connection state

## Phase 2: Workout Sync + Exercise Template Cache [DONE]

- `app/storage/workout.go` — `workouts`, `workout_exercises`, `workout_sets` tables
- `app/storage/exercise_cache.go` — `exercise_templates` table (caches Hevy templates for analysis)
- `app/storage/sync.go` — `sync_state` table (tracks last sync per user)
- `/init` — full sync: fetches all exercise templates + all workouts from Hevy, saves locally
- `/sync` — incremental sync: fetches latest page, saves new workouts
- `/last` — fetches latest workout from Hevy, displays rich summary with volume comparison
- `/start` menu adapts: not synced → "Run Initial Sync", synced → "Last Workout" + "Analyze"

## Phase 3: Analysis Engine [DONE]

Package `app/analysis/` — pure Go, no external API:
- `volume.go` — weekly sets & tonnage per muscle group (primary: 1.0, secondary: 0.5 credit, excludes warmup)
- `progressive.go` — estimated 1RM trends via Epley formula, detects progressing/stalling/regressing
- `balance.go` — push/pull/legs ratios, upper:lower ratio, flags imbalances
- `frequency.go` — workouts/week, session duration, muscle frequency, rest days
- `engine.go` — coordinator + Telegram-friendly formatters
- `/analyze` command — inline keyboard: Volume Report / Progressive Overload / Muscle Balance / Full Report

---

## Phase 4: LLM Integration + Smart Recommendations [TODO]

### New package: `app/llm/`

- `client.go` — LLM API client (Claude/OpenAI), env vars: `LLM_PROVIDER`, `LLM_API_KEY`, `LLM_MODEL`
- `prompts.go` — system prompt (strength coach persona), analysis prompt, template generation prompt

### Enhanced `/analyze`
After algorithmic analysis, pipe results through LLM for natural language interpretation + top 3 recommendations.

### `/advice` command
Contextual recommendations referencing specific exercises/muscles from latest analysis.

### Graceful fallback
If no LLM key configured, show algorithmic-only results (Phase 3 output).

---

## Phase 5: Template Generation + Push to Hevy [TODO]

### `/template` command
Inline keyboard: Push Day / Pull Day / Leg Day / Upper Body / Lower Body / Custom

### Flow
1. Run analysis on recent data
2. Fetch existing routines (avoid duplication)
3. Send to LLM with template generation prompt
4. LLM returns structured JSON (exercise IDs, sets, reps, RPE)
5. Bot validates exercise IDs against cache
6. Preview → [Push to Hevy] / [Regenerate] / [Discard]
7. On push: `POST /v1/routines` → routine appears in Hevy app

---

## Environment Variables

```
TELEGRAM_TOKEN=...        # Telegram bot token (required)
DATA_FILE_PATH=data.db    # SQLite database path (required)
HEVY_API_KEY=...          # Per-user keys are primary, this is for backwards compat
LLM_PROVIDER=claude       # Phase 4: claude or openai
LLM_API_KEY=...           # Phase 4: LLM API key
LLM_MODEL=...             # Phase 4: model identifier
```
