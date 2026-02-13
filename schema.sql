CREATE TABLE IF NOT EXISTS users
(
    telegram_id INTEGER PRIMARY KEY NOT NULL
);

CREATE TABLE IF NOT EXISTS user_exercises
(
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER NOT NULL,
    exercise_id       TEXT    NOT NULL,
    exercise_name     TEXT    NOT NULL,
    exercise_category TEXT    NOT NULL,
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (telegram_id)
);

CREATE INDEX IF NOT EXISTS idx_user_exercises_user_id ON user_exercises (user_id);

CREATE TABLE IF NOT EXISTS workouts
(
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER  NOT NULL,
    workout_name  TEXT     NOT NULL,
    workout_date  DATETIME NOT NULL,
    duration      TEXT,
    raw_text      TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (telegram_id)
);

CREATE INDEX IF NOT EXISTS idx_workouts_user_id ON workouts (user_id);
CREATE INDEX IF NOT EXISTS idx_workouts_user_date ON workouts (user_id, workout_date);

CREATE TABLE IF NOT EXISTS workout_sets
(
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    workout_id    INTEGER NOT NULL,
    exercise_name TEXT    NOT NULL,
    set_order     INTEGER NOT NULL,
    weight        REAL    DEFAULT 0,
    reps          INTEGER DEFAULT 0,
    distance      REAL    DEFAULT 0,
    seconds       INTEGER DEFAULT 0,
    notes         TEXT,
    FOREIGN KEY (workout_id) REFERENCES workouts (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workout_sets_workout_id ON workout_sets (workout_id);
