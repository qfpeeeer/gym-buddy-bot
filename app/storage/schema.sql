-- Users
CREATE TABLE IF NOT EXISTS users (
    telegram_id  INTEGER PRIMARY KEY NOT NULL,
    hevy_api_key TEXT
);

-- Workouts (synced from Hevy)
CREATE TABLE IF NOT EXISTS workouts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    hevy_id    TEXT NOT NULL,
    title      TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time   DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(telegram_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_workouts_hevy_id ON workouts(hevy_id);
CREATE INDEX IF NOT EXISTS idx_workouts_user_id ON workouts(user_id);

-- Workout exercises
CREATE TABLE IF NOT EXISTS workout_exercises (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    workout_id           INTEGER NOT NULL,
    exercise_index       INTEGER NOT NULL,
    title                TEXT NOT NULL,
    exercise_template_id TEXT NOT NULL,
    notes                TEXT,
    superset_id          INTEGER,
    FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE
);

-- Workout sets
CREATE TABLE IF NOT EXISTS workout_sets (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_id INTEGER NOT NULL,
    set_index   INTEGER NOT NULL,
    set_type    TEXT DEFAULT 'normal',
    weight_kg   REAL,
    reps        INTEGER,
    distance_m  REAL,
    duration_s  INTEGER,
    rpe         REAL,
    FOREIGN KEY (exercise_id) REFERENCES workout_exercises(id) ON DELETE CASCADE
);

-- Hevy exercise template cache (for analysis muscle group lookups)
CREATE TABLE IF NOT EXISTS exercise_templates (
    id                      TEXT PRIMARY KEY,
    title                   TEXT NOT NULL,
    type                    TEXT NOT NULL,
    primary_muscle_group    TEXT,
    secondary_muscle_groups TEXT,
    equipment               TEXT,
    is_custom               INTEGER DEFAULT 0,
    fetched_at              DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sync state (per-user)
CREATE TABLE IF NOT EXISTS sync_state (
    user_id        INTEGER PRIMARY KEY,
    last_sync      DATETIME,
    total_workouts INTEGER DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(telegram_id)
);
