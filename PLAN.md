# GymBuddy Bot: Hevy Integration + AI Coach

## Context

The user has been tracking workouts in the Strong app for 5 months (PPL/ULUL split, 5-6 templates). Strong has no public API, making it impossible to build smart features on top of it. Hevy has an official REST API (verified Feb 2026) and the user has already obtained an API key. The goal is to transform the bot from a simple import tool into an **AI-powered training coach** that reads workout data from Hevy, analyzes training patterns, and creates optimized templates.

**Core principle:** Hevy = gym logging UI, Telegram bot = smart coach layer (analysis, recommendations, template creation).

---

## Phase 1: Hevy API Client + User Connection [DONE]

### New package: `app/hevy/`

**`app/hevy/client.go`** — HTTP client wrapper:
- Constructor takes API key, returns `*Client`
- All methods accept `context.Context`
- Handles pagination internally (auto-fetch all pages)
- Rate limiting: 200ms delay between pages, exponential backoff on 429
- Common error types: `ErrUnauthorized`, `ErrRateLimited`, `ErrNotFound`

**`app/hevy/models.go`** — Hevy API types:
```go
type Workout struct {
    ID, Title, Description string
    StartTime, EndTime     time.Time
    Exercises              []WorkoutExercise
    CreatedAt, UpdatedAt   time.Time
}
type WorkoutExercise struct {
    Index              int
    Title, Notes       string
    ExerciseTemplateID string
    SupersetID         *int
    Sets               []WorkoutSet
}
type WorkoutSet struct {
    Index           int
    Type            string   // normal, warmup, dropset, failure
    WeightKG        *float64
    Reps            *int
    DistanceMeters  *float64
    DurationSeconds *int
    RPE             *float64
}
type ExerciseTemplate struct {
    ID, Title, Type        string
    PrimaryMuscleGroup     string
    SecondaryMuscleGroups  []string
    Equipment              string
    IsCustom               bool
}
type Routine struct {
    ID, Title string
    FolderID  *string
    Exercises []RoutineExercise
}
```

**`app/hevy/workouts.go`**, **`app/hevy/exercises.go`**, **`app/hevy/routines.go`** — endpoint methods:
- `GetWorkouts(page, pageSize)`, `GetAllWorkouts(progressCb)`, `GetWorkoutByID(id)`
- `GetWorkoutCount()`, `GetWorkoutEvents(page, pageSize)`
- `GetExerciseTemplates(page, pageSize)`, `GetAllExerciseTemplates()`
- `GetRoutines(page, pageSize)`, `CreateRoutine(req)`, `UpdateRoutine(id, req)`
- `CreateRoutineFolder(title)`

### DB changes

**`app/storage/user.go`** — add columns to users table:
```sql
ALTER TABLE users ADD COLUMN hevy_api_key TEXT;
```
New methods: `SetHevyAPIKey(userID, key)`, `GetHevyAPIKey(userID)`, `ClearHevyAPIKey(userID)`

### Bot commands

**`/connect`** — prompts user to send their Hevy API key:
1. Bot sends: "Send me your Hevy API key (get it from hevy.com/settings > Developer)."
2. Sets a per-user state "awaiting_hevy_key" (in-memory map in MessageHandler)
3. User pastes key -> MessageHandler intercepts
4. Bot validates key with `GetWorkoutCount()`
5. On success: stores key, deletes user's message (security), replies with workout count
6. On failure: reports error, lets user retry

**`/disconnect`** — clears stored API key

### Files modified
- `app/storage/user.go` — added `hevy_api_key` column + methods
- `app/user/user.go` — added Storage interface methods + HevyClient(userID) factory
- `app/events/events.go` — expanded UserManager interface
- `app/events/command_handler.go` — added `/connect`, `/disconnect`
- `app/events/message_handler.go` — added "awaiting key" state handling
- `app/events/callback_query_handler.go` — added `connect_hevy` button handler
- `app/main.go` — wired SetAwaitingHevyKey callback between handlers

---

## Phase 2: Workout Sync + Exercise Template Cache

### New storage

**`app/storage/hevy_sync.go`** — sync state tracking:
```sql
CREATE TABLE IF NOT EXISTS hevy_sync_state (
    user_id          INTEGER PRIMARY KEY,
    last_sync        DATETIME,
    total_workouts   INTEGER DEFAULT 0,
    sync_complete    INTEGER DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(telegram_id)
);
```
Methods: `GetSyncState(userID)`, `UpdateSyncState(userID, state)`

**`app/storage/hevy_exercise_cache.go`** — exercise template cache:
```sql
CREATE TABLE IF NOT EXISTS hevy_exercise_templates (
    id                      TEXT PRIMARY KEY,
    title                   TEXT NOT NULL,
    type                    TEXT NOT NULL,
    primary_muscle_group    TEXT,
    secondary_muscle_groups TEXT,  -- JSON array
    equipment               TEXT,
    is_custom               INTEGER DEFAULT 0,
    fetched_at              DATETIME DEFAULT CURRENT_TIMESTAMP
);
```
Methods: `UpsertExerciseTemplates(templates)`, `GetExerciseTemplate(id)`, `GetAllExerciseTemplates()`, `GetTemplatesByMuscle(muscle)`

### Workout storage extension

**`app/storage/workout.go`** — add Hevy fields:
```sql
ALTER TABLE workouts ADD COLUMN hevy_id TEXT;
ALTER TABLE workouts ADD COLUMN end_time DATETIME;
ALTER TABLE workouts ADD COLUMN source TEXT DEFAULT 'strong';
CREATE UNIQUE INDEX IF NOT EXISTS idx_workouts_hevy_id ON workouts(hevy_id);

ALTER TABLE workout_sets ADD COLUMN exercise_template_id TEXT;
ALTER TABLE workout_sets ADD COLUMN set_type TEXT DEFAULT 'normal';
ALTER TABLE workout_sets ADD COLUMN rpe REAL;
```
New methods: `SaveHevyWorkout(userID, hevy.Workout)`, `WorkoutExistsByHevyID(hevyID)`, `GetAllWorkouts(userID)`

### Bot commands

**`/init`** — full historical sync:
1. Check connection
2. Fetch workout count, send "Importing N workouts..."
3. Fetch all exercise templates -> cache locally
4. Paginate all workouts -> save each (skip if `hevy_id` already exists)
5. Send progress updates every 20 workouts
6. On completion: "Synced N workouts, M exercises cached. Run /analyze for insights."

**`/sync`** — incremental sync:
1. Fetch page 1 of workouts
2. Save any new ones (check by `hevy_id`)
3. Report how many new workouts found

**`/last`** — fetch & display latest workout:
1. Fetch page 1 (pageSize=1) from Hevy
2. Save if new
3. Format rich summary: title, date, duration, exercises with sets/weight/reps
4. Compare volume to previous session with same title (if exists)

### Updated `/start` menu

Adapt based on user state:
- Not connected -> "Connect Hevy" button + existing buttons
- Connected, not synced -> "Run Initial Setup" + existing
- Connected + synced -> "Fetch Latest", "Analyze", "My Routines" + existing

### Files to create
- `app/storage/hevy_sync.go`, `app/storage/hevy_exercise_cache.go`

### Files to modify
- `app/storage/workout.go` — add columns, `SaveHevyWorkout`, `WorkoutExistsByHevyID`
- `app/user/user.go` — add sync orchestration methods
- `app/events/command_handler.go` — add `/init`, `/sync`, `/last`
- `app/events/events.go` — expand interface

### Verification
- `/init` — import all workouts, verify count matches Hevy
- `/sync` — log a workout in Hevy, run /sync, verify it appears
- `/last` — verify latest workout is displayed with full detail
- Run `go test ./...` — existing Strong tests must still pass

---

## Phase 3: Analysis Engine (Pure Go, No External API)

### New package: `app/analysis/`

**`app/analysis/models.go`** — result types:
```go
type AnalysisResult struct {
    Period         string // e.g. "Last 4 weeks"
    VolumeReport   VolumeReport
    OverloadReport OverloadReport
    BalanceReport  BalanceReport
    FrequencyReport FrequencyReport
}
type MuscleVolume struct {
    Muscle       string
    WeeklySets   float64 // primary sets + 0.5*secondary sets
    WeeklyTonnage float64 // sum(weight * reps)
    Status       string  // "low", "optimal", "high"
}
type ExerciseProgress struct {
    ExerciseName string
    E1RMTrend    []float64 // estimated 1RM per session
    Trend        string    // "progressing", "stalling", "regressing"
    BestE1RM     float64
    LatestE1RM   float64
}
```

**`app/analysis/volume.go`** — weekly volume per muscle group:
- Count working sets per muscle (exclude warmup sets via `set_type`)
- Primary muscle: 1.0 set credit, secondary: 0.5
- Uses cached exercise templates for muscle group mapping
- Flags: <10 sets/week = "low", 10-20 = "optimal", >20 = "high"
- Weekly tonnage: sum(weight_kg * reps) per muscle

**`app/analysis/progressive.go`** — progressive overload tracking:
- Compute estimated 1RM per exercise per session: `e1RM = weight * (1 + reps/30)` (Epley)
- Compare last 4 weeks vs previous 4 weeks
- Detect: stalling (no e1RM increase in 3+ sessions), regression, PRs

**`app/analysis/balance.go`** — push/pull/legs ratios:
- Classify exercises by primary_muscle_group into Push/Pull/Legs
- Compute Push:Pull ratio (target ~1:1), Upper:Lower ratio
- Flag imbalances

**`app/analysis/frequency.go`** — training patterns:
- Workouts/week, sessions/week per muscle group
- Average session duration
- Rest day distribution

**`app/analysis/engine.go`** — coordinator:
- `Analyze(workouts, templates, period) -> AnalysisResult`
- Combines all sub-analyzers
- Formats result as structured text for both display and LLM input

### Bot command

**`/analyze`** — inline keyboard with options:
- [Volume Report] — muscle group volume breakdown
- [Progressive Overload] — exercise progress trends
- [Muscle Balance] — push/pull/upper/lower ratios
- [Full Report] — all of the above combined

Each option sends a formatted Telegram message with the analysis. No LLM yet — pure algorithmic results.

### Files to create
- `app/analysis/engine.go`, `models.go`, `volume.go`, `progressive.go`, `balance.go`, `frequency.go`

### Files to modify
- `app/user/user.go` — add `RunAnalysis(userID, period)` method
- `app/events/command_handler.go` — add `/analyze`
- `app/events/callback_query_handler.go` — add analysis sub-menu callbacks
- `app/events/events.go` — expand interface

### Verification
- Import workouts via `/init`, run `/analyze` -> [Full Report]
- Verify volume numbers make sense for PPL split
- Verify progressive overload detects known PRs
- `go test ./app/analysis/...` — unit tests with fixture data

---

## Phase 4: LLM Integration + Smart Recommendations

### New package: `app/llm/`

**`app/llm/client.go`** — LLM API client:
- Support Claude (Anthropic API) and OpenAI as providers
- Env vars: `LLM_PROVIDER` (claude/openai), `LLM_API_KEY`, `LLM_MODEL`
- Simple `Complete(systemPrompt, userPrompt) -> string` method
- Timeout, retry, token limit handling

**`app/llm/prompts.go`** — prompt templates:
- System prompt: strength training coach persona, concise Telegram-friendly output
- Analysis prompt: takes `AnalysisResult` as structured data, asks for interpretation + recommendations
- Template generation prompt: takes analysis + existing routines + available exercises, returns structured JSON routine

### Enhanced `/analyze`

After algorithmic analysis, pipe results through LLM:
1. Run analysis engine (Phase 3)
2. Serialize results as text
3. Send to LLM with analysis prompt
4. LLM returns: key observations, strengths, weaknesses, top 3 recommendations
5. Bot sends both: data summary + AI interpretation

### New: `/advice` (or button after `/analyze`)

Contextual recommendations based on latest analysis:
- "Your chest volume is high (22 sets/week) but rear delts are neglected (4 sets/week). Consider replacing 2 chest sets with face pulls."
- "Bench press has stalled for 3 weeks. Try adding a paused bench variation or adjust rep scheme."

### Files to create
- `app/llm/client.go`, `prompts.go`, `models.go`

### Files to modify
- `app/main.go` — init LLM client from env vars, pass to UserManager
- `app/user/user.go` — add LLM-enhanced analysis method
- `app/events/command_handler.go` — enhance `/analyze` output
- `deployments/.env` / `example.env` — add LLM env vars

### Verification
- Run `/analyze` [Full Report] — verify LLM summary appears after data
- Verify recommendations are actionable and reference specific exercises/muscles
- Test with missing LLM key — bot should gracefully fall back to algorithmic-only analysis

---

## Phase 5: Template Generation + Push to Hevy

### Bot command

**`/template`** — inline keyboard:
- [Suggest Push Day] [Suggest Pull Day] [Suggest Leg Day]
- [Suggest Upper Body] [Suggest Lower Body]
- [Custom — describe what you want]

### Template generation flow

1. User selects template type (e.g. "Push Day")
2. Bot runs analysis on recent data (last 4 weeks)
3. Fetches user's existing Hevy routines (to avoid duplication)
4. Fetches exercise templates filtered by relevant muscle groups
5. Sends everything to LLM with template generation prompt
6. LLM returns structured JSON: exercise list with template IDs, set counts, rep ranges, RPE targets
7. Bot validates all exercise_template_ids exist in cache
8. Bot shows preview message to user:
   ```
   Suggested Push Day:
   1. Bench Press (Barbell) — 4x6-8 @RPE 8
   2. Incline Dumbbell Press — 3x8-10 @RPE 7
   3. Cable Lateral Raise — 3x12-15
   4. Tricep Pushdown — 3x10-12
   5. Overhead Tricep Extension — 2x12-15
   
   [Push to Hevy] [Regenerate] [Discard]
   ```
9. On "Push to Hevy": calls `POST /v1/routines` -> routine appears in Hevy app
10. On "Regenerate": re-runs LLM with note "different from previous suggestion"

### "Custom" option
User types free-text description: "I want a back-focused pull day with extra bicep work"
Bot sends this + analysis context to LLM, same flow.

### Files to modify
- `app/hevy/routines.go` — `CreateRoutine` already planned
- `app/llm/prompts.go` — add template generation prompt
- `app/events/command_handler.go` — add `/template`
- `app/events/callback_query_handler.go` — template type selection + push/discard callbacks
- `app/events/message_handler.go` — handle free-text template description (state-based)

### Verification
- Generate a Push Day template, verify exercise IDs are valid
- Push to Hevy, verify routine appears in the Hevy app
- Test "Regenerate" produces different output
- Test "Custom" with free-text description

---

## Implementation Order Summary

| Phase | What | Depends On | Key Value | Status |
|-------|------|------------|-----------|--------|
| 1 | Hevy client + `/connect` | Nothing | Foundation — user can link account | DONE |
| 2 | Sync + `/init` + `/last` | Phase 1 | Data — all workout history in bot's DB | TODO |
| 3 | Analysis engine | Phase 2 | Insights — volume, overload, balance reports | TODO |
| 4 | LLM integration | Phase 3 | Intelligence — natural language recommendations | TODO |
| 5 | Template generation | Phase 4 | Action — create routines from AI suggestions | TODO |

Each phase is independently deployable and testable. Phase 3 provides value even without LLM (pure algorithmic analysis). Phase 4-5 can use any LLM provider.

---

## New Environment Variables

```
HEVY_API_KEY=...          # (kept for backwards compat, but per-user keys are primary)
LLM_PROVIDER=claude       # claude or openai
LLM_API_KEY=sk-ant-...
LLM_MODEL=claude-sonnet-4-5-20250929
```
