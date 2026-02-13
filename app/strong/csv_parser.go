package strong

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// csvDateFormats to try when parsing the Date column.
var csvDateFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

// ParseCSV parses a Strong app CSV export into a slice of Workouts.
// Each row in the CSV represents one set. Rows are grouped by (Date, Workout Name)
// into Workout structs, and by Exercise Name within each workout.
func ParseCSV(reader io.Reader) ([]Workout, error) {
	r := csv.NewReader(reader)

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.TrimSpace(col)] = i
	}

	// verify required columns exist
	required := []string{"Date", "Workout Name", "Exercise Name", "Set Order", "Weight", "Reps"}
	for _, col := range required {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("missing required column %q in CSV header", col)
		}
	}

	type workoutKey struct {
		date time.Time
		name string
	}

	// preserve insertion order
	var workoutOrder []workoutKey
	workoutMap := make(map[workoutKey]*Workout)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		dateStr := getCol(record, colIndex, "Date")
		workoutName := getCol(record, colIndex, "Workout Name")
		exerciseName := getCol(record, colIndex, "Exercise Name")
		setOrderStr := getCol(record, colIndex, "Set Order")
		weightStr := getCol(record, colIndex, "Weight")
		repsStr := getCol(record, colIndex, "Reps")
		duration := getCol(record, colIndex, "Duration")
		distanceStr := getCol(record, colIndex, "Distance")
		secondsStr := getCol(record, colIndex, "Seconds")
		notes := getCol(record, colIndex, "Notes")

		parsedDate, err := parseCSVDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date %q: %w", dateStr, err)
		}

		key := workoutKey{date: parsedDate, name: workoutName}

		workout, exists := workoutMap[key]
		if !exists {
			workout = &Workout{
				Name:     workoutName,
				Date:     parsedDate,
				Duration: duration,
			}
			workoutMap[key] = workout
			workoutOrder = append(workoutOrder, key)
		}

		setOrder, _ := strconv.Atoi(setOrderStr)
		weight, _ := strconv.ParseFloat(strings.ReplaceAll(weightStr, ",", "."), 64)
		reps, _ := strconv.Atoi(repsStr)
		distance, _ := strconv.ParseFloat(distanceStr, 64)
		seconds, _ := strconv.Atoi(secondsStr)

		set := ExerciseSet{
			SetOrder: setOrder,
			Weight:   weight,
			Reps:     reps,
			Distance: distance,
			Seconds:  seconds,
		}

		// find or create the exercise within this workout
		found := false
		for i := range workout.Exercises {
			if workout.Exercises[i].Name == exerciseName {
				workout.Exercises[i].Sets = append(workout.Exercises[i].Sets, set)
				if notes != "" && workout.Exercises[i].Notes == "" {
					workout.Exercises[i].Notes = notes
				}
				found = true
				break
			}
		}
		if !found {
			workout.Exercises = append(workout.Exercises, WorkoutExercise{
				Name:  exerciseName,
				Notes: notes,
				Sets:  []ExerciseSet{set},
			})
		}
	}

	// build result in order
	result := make([]Workout, 0, len(workoutOrder))
	for _, key := range workoutOrder {
		result = append(result, *workoutMap[key])
	}

	return result, nil
}

// getCol safely retrieves a column value from a record.
func getCol(record []string, colIndex map[string]int, col string) string {
	idx, ok := colIndex[col]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

// parseCSVDate tries multiple date formats for the CSV Date column.
func parseCSVDate(dateStr string) (time.Time, error) {
	for _, format := range csvDateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized CSV date format: %q", dateStr)
}
