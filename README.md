# gym-buddy-bot

GymBuddy is a Telegram bot designed to be your virtual fitness companion, helping you track gym exercises and receive
personalized workout recommendations. It provides exercise guidance and motivation directly in your Telegram chat.

## Core Features

### 1. Google Sheets Integration

- Users connect their Google Sheets upon `/start`
- Bot tracks weights, exercises, and reps in the user's sheet
- TODO: Research Google Sheets API integration and authentication

### 2. Main Menu

#### 2.1 Get Today's Exercises

1. **Change**
    - Change rep count
    - Change suggested weight
    - Change the exercise
        - Suggest similar
        - Search
2. **Start Today's Session**

#### 2.2 Logging Today's Exercises

- Display exercises as `[Exercise Name X/Y]` (e.g., `[Push ups 2/4]`)
- Show exercise description from `exercises.json`
- For each exercise:
    1. `[✅ Completed rep]`
    2. `[🔄 Completed with different weight]`
    3. `[🔶 Partially completed rep]`

#### 2.3 View Progress

- Display charts and statistics
- Show streaks and achievements
- TODO: Research Google Sheets API integration and authentication for data visualization

#### 2.4 Settings

- Change Google Sheets
- Change workout plan
- TODO: Research Google Sheets API integration and authentication