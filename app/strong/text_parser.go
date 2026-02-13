package strong

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// setLineRegex matches lines like "Set 1: 20 kg × 12" or "Set 1: 12" (bodyweight)
	setLineRegex = regexp.MustCompile(`^Set\s+(\d+):\s+(.+)$`)

	// weightRepsRegex matches "20 kg × 12", "17,5 kg x 12", "22.5 kg × 10"
	weightRepsRegex = regexp.MustCompile(`^([\d.,]+)\s*kg\s*[x×]\s*(\d+)$`)

	// repsOnlyRegex matches just a number (bodyweight exercises)
	repsOnlyRegex = regexp.MustCompile(`^(\d+)$`)

	// notesRegex matches "Notes: some text"
	notesRegex = regexp.MustCompile(`^Notes:\s*(.+)$`)

	// urlRegex matches lines that are just a URL (e.g. Strong app share links)
	urlRegex = regexp.MustCompile(`^https?://\S+$`)

	// dateFormats to try when parsing the date line
	dateFormats = []string{
		"Monday, 2 January 2006 at 15:04",
		"Monday, 02 January 2006 at 15:04",
		"Monday, January 2, 2006 at 15:04",
		"Monday, January 2, 2006 at 3:04 PM",
		"2 January 2006 at 15:04",
		"2006-01-02 15:04:05",
	}
)

// ParseText parses the Strong app's share-text format into a Workout.
func ParseText(text string) (*Workout, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	if len(lines) < 3 {
		return nil, fmt.Errorf("text too short to be a workout")
	}

	workoutName := strings.TrimSpace(lines[0])
	if workoutName == "" {
		return nil, fmt.Errorf("empty workout name")
	}

	dateLine := strings.TrimSpace(lines[1])
	workoutDate, err := parseDate(dateLine)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date %q: %w", dateLine, err)
	}

	workout := &Workout{
		Name:    workoutName,
		Date:    workoutDate,
		RawText: text,
	}

	var currentExercise *WorkoutExercise

	for i := 2; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if line == "" {
			continue
		}

		// skip URL lines (e.g. Strong app share links)
		if urlRegex.MatchString(line) {
			continue
		}

		// check if it's a notes line
		if match := notesRegex.FindStringSubmatch(line); match != nil {
			if currentExercise != nil {
				currentExercise.Notes = match[1]
			}
			continue
		}

		// check if it's a set line
		if match := setLineRegex.FindStringSubmatch(line); match != nil {
			if currentExercise == nil {
				return nil, fmt.Errorf("set line without exercise at line %d: %q", i+1, line)
			}

			setOrder, _ := strconv.Atoi(match[1])
			set, err := parseSetData(match[2], setOrder)
			if err != nil {
				return nil, fmt.Errorf("failed to parse set at line %d: %w", i+1, err)
			}
			currentExercise.Sets = append(currentExercise.Sets, set)
			continue
		}

		// otherwise it's an exercise name — save previous and start new
		if currentExercise != nil {
			workout.Exercises = append(workout.Exercises, *currentExercise)
		}
		currentExercise = &WorkoutExercise{Name: line}
	}

	// save the last exercise
	if currentExercise != nil {
		workout.Exercises = append(workout.Exercises, *currentExercise)
	}

	if len(workout.Exercises) == 0 {
		return nil, fmt.Errorf("no exercises found in workout text")
	}

	return workout, nil
}

// parseDate tries multiple date formats to parse the date line.
func parseDate(dateLine string) (time.Time, error) {
	for _, format := range dateFormats {
		if t, err := time.Parse(format, dateLine); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", dateLine)
}

// parseSetData parses the data portion of a set line (after "Set N: ").
func parseSetData(data string, setOrder int) (ExerciseSet, error) {
	data = strings.TrimSpace(data)

	// try weight × reps format (e.g., "20 kg × 12", "17,5 kg x 12")
	if match := weightRepsRegex.FindStringSubmatch(data); match != nil {
		weight, err := parseWeight(match[1])
		if err != nil {
			return ExerciseSet{}, fmt.Errorf("failed to parse weight %q: %w", match[1], err)
		}
		reps, _ := strconv.Atoi(match[2])
		return ExerciseSet{SetOrder: setOrder, Weight: weight, Reps: reps}, nil
	}

	// try reps only (bodyweight exercises)
	if match := repsOnlyRegex.FindStringSubmatch(data); match != nil {
		reps, _ := strconv.Atoi(match[1])
		return ExerciseSet{SetOrder: setOrder, Reps: reps}, nil
	}

	return ExerciseSet{}, fmt.Errorf("unrecognized set format: %q", data)
}

// parseWeight handles both "20" and "17,5" (European decimal comma) weight formats.
func parseWeight(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}
