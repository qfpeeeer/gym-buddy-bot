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

CREATE TABLE IF NOT EXISTS user_google_sheets
(
    user_id       INTEGER PRIMARY KEY NOT NULL,
    access_token  TEXT                NOT NULL,
    refresh_token TEXT                NOT NULL,
    token_type    TEXT                NOT NULL,
    expiry        DATETIME            NOT NULL,
    sheet_id      TEXT,
    FOREIGN KEY (user_id) REFERENCES users (telegram_id)
);

CREATE TABLE IF NOT EXISTS user_states
(
    user_id INTEGER PRIMARY KEY NOT NULL,
    state   TEXT,
    FOREIGN KEY (user_id) REFERENCES users (telegram_id)
);