// Script to import Strong CSV workouts into Hevy via API.
//
// Usage: go run scripts/import_to_hevy.go -csv <path_to_csv> -key <hevy_api_key> [-dry-run]
//
// Steps:
// 1. Fetches all Hevy exercise templates
// 2. Maps Strong exercise names to Hevy template IDs
// 3. Groups CSV rows into workouts
// 4. Creates each workout via POST /v1/workouts
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Hevy API types (minimal for this script)
type HevyExerciseTemplate struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type HevyTemplatesResponse struct {
	Page              int                    `json:"page"`
	PageCount         int                    `json:"page_count"`
	ExerciseTemplates []HevyExerciseTemplate `json:"exercise_templates"`
}

type HevyWorkoutRequest struct {
	Workout HevyWorkoutBody `json:"workout"`
}

type HevyWorkoutBody struct {
	Title       string             `json:"title"`
	Description string             `json:"description,omitempty"`
	IsPrivate   bool               `json:"is_private"`
	StartTime   string             `json:"start_time"`
	EndTime     string             `json:"end_time"`
	Exercises   []HevyExerciseBody `json:"exercises"`
}

type HevyExerciseBody struct {
	ExerciseTemplateID string        `json:"exercise_template_id"`
	Notes              string        `json:"notes,omitempty"`
	Sets               []HevySetBody `json:"sets"`
}

type HevySetBody struct {
	Type            string   `json:"type"`
	WeightKG        *float64 `json:"weight_kg,omitempty"`
	Reps            *int     `json:"reps,omitempty"`
	DistanceMeters  *float64 `json:"distance_meters,omitempty"`
	DurationSeconds *int     `json:"duration_seconds,omitempty"`
	RPE             *float64 `json:"rpe,omitempty"`
}

// Strong CSV row
type StrongRow struct {
	Date         string
	WorkoutName  string
	Duration     string
	ExerciseName string
	SetOrder     string
	Weight       float64
	Reps         int
	Distance     float64
	Seconds      int
	Notes        string
	WorkoutNotes string
	RPE          float64
}

// Parsed workout
type Workout struct {
	Name      string
	Date      time.Time
	Duration  string
	Notes     string
	Exercises []Exercise
}

type Exercise struct {
	Name  string
	Notes string
	Sets  []Set
}

type Set struct {
	Weight  float64
	Reps    int
	RPE     float64
	Seconds int
}

// Manual mapping for exercises whose Strong names don't match Hevy titles.
var manualMapping = map[string]string{
	// Strong name -> Hevy title (will be resolved to ID at runtime)
	"Back Extension":                          "Back Extension (Hyperextension)",
	"Bent Over One Arm Row (Dumbbell)":        "Dumbbell Row",
	"Cable Crossover":                         "Single Arm Cable Crossover",
	"Chest Fly":                               "Chest Fly (Machine)",
	"Face Pull (Cable)":                       "Face Pull",
	"Goblet Squat (Kettlebell)":               "Goblet Squat",
	"Hip Adductor (Machine)":                  "Hip Adduction (Machine)",
	"Lat Pulldown - Wide Grip (Cable)":        "Lat Pulldown (Cable)",
	"Lat Pulldown (Single Arm)":               "Single Arm Lat Pulldown",
	"Leg Press":                               "Leg Press (Machine)",
	"Overead Press Machine":                   "Shoulder Press (Machine Plates)",
	"Pec Deck (Machine)":                      "Butterfly (Pec Deck)",
	"Pull Up (Narrow Grip)":                   "Pull Up",
	"Pullover Cable (for Back)":               "Straight Arm Lat Pulldown (Cable)",
	"Reverse Fly (Cable)":                     "Rear Delt Reverse Fly (Cable)",
	"Reverse Fly (Machine)":                   "Rear Delt Reverse Fly (Machine)",
	"Seated Row (Cable)":                      "Seated Cable Row - V Grip (Cable)",
	"Shoulder Press (Machine)":                "Shoulder Press (Machine Plates)",
	"Shoulder Press (Plate Loaded)":           "Shoulder Press (Machine Plates)",
	"Triceps Extension":                       "Triceps Extension (Cable)",
	"Triceps Extenstion":                      "Triceps Extension (Cable)",
	"Triceps Isolated Cable Pull (One Hand)":  "Single Arm Triceps Pushdown (Cable)",
	"Triceps Pushdown (Cable - Straight Bar)": "Triceps Pushdown",
	"Французский Жим Гантелями":               "Skullcrusher (Dumbbell)",
}

// Exercises that need to be created as custom in Hevy before import.
var customExercises = map[string]struct{}{
	"X-machine": {},
	"Вращение Блина Вокруг Головы": {},
}

const baseURL = "https://api.hevyapp.com/v1"

var apiKey string

func main() {
	csvPath := flag.String("csv", "", "path to Strong CSV export")
	key := flag.String("key", "", "Hevy API key")
	dryRun := flag.Bool("dry-run", false, "print what would be done without calling API")
	flag.Parse()

	if *csvPath == "" || *key == "" {
		fmt.Println("Usage: go run scripts/import_to_hevy.go -csv <path> -key <hevy_api_key> [-dry-run]")
		os.Exit(1)
	}
	apiKey = *key

	// 1. Fetch all Hevy exercise templates
	fmt.Println("Fetching Hevy exercise templates...")
	templates, err := fetchAllTemplates()
	if err != nil {
		fmt.Printf("Error fetching templates: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Fetched %d exercise templates\n", len(templates))

	// Build title -> ID map (lowercase for matching)
	titleToID := make(map[string]string)
	for _, t := range templates {
		titleToID[strings.ToLower(strings.TrimSpace(t.Title))] = t.ID
	}

	// 2. Parse CSV
	fmt.Println("Parsing CSV...")
	workouts, err := parseCSV(*csvPath)
	if err != nil {
		fmt.Printf("Error parsing CSV: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Parsed %d workouts\n", len(workouts))

	// 3. Resolve exercise name -> template ID
	unmapped := make(map[string]bool)
	resolveExercise := func(name string) string {
		trimmed := strings.TrimSpace(name)

		// Check manual mapping first
		if mapped, ok := manualMapping[trimmed]; ok {
			trimmed = mapped
		}

		// Direct match
		if id, ok := titleToID[strings.ToLower(trimmed)]; ok {
			return id
		}

		// Try without trailing space
		if id, ok := titleToID[strings.ToLower(strings.TrimRight(trimmed, " "))]; ok {
			return id
		}

		unmapped[trimmed] = true
		return ""
	}

	// Pre-scan for unmapped exercises
	for _, w := range workouts {
		for _, ex := range w.Exercises {
			resolveExercise(ex.Name)
		}
	}

	if len(unmapped) > 0 {
		fmt.Printf("\n--- UNMAPPED EXERCISES (%d) ---\n", len(unmapped))
		for name := range unmapped {
			fmt.Printf("  - %q\n", name)
		}
		fmt.Println("\nThese exercises will be SKIPPED during import.")
		fmt.Println("Add them to manualMapping or customExercises in the script to include them.")
		fmt.Println()
	}

	// 4. Import workouts
	// Sort by date (oldest first)
	sort.Slice(workouts, func(i, j int) bool {
		return workouts[i].Date.Before(workouts[j].Date)
	})

	imported := 0
	skipped := 0
	for i, w := range workouts {
		// Build Hevy exercises
		var hevyExercises []HevyExerciseBody
		for _, ex := range w.Exercises {
			templateID := resolveExercise(ex.Name)
			if templateID == "" {
				continue
			}

			var sets []HevySetBody
			for _, s := range ex.Sets {
				set := HevySetBody{Type: "normal"}
				if s.Weight > 0 {
					w := s.Weight
					set.WeightKG = &w
				}
				if s.Reps > 0 {
					r := s.Reps
					set.Reps = &r
				}
				if s.RPE > 0 {
					rpe := s.RPE
					set.RPE = &rpe
				}
				if s.Seconds > 0 {
					sec := s.Seconds
					set.DurationSeconds = &sec
				}
				sets = append(sets, set)
			}

			if len(sets) == 0 {
				continue
			}

			hevyExercises = append(hevyExercises, HevyExerciseBody{
				ExerciseTemplateID: templateID,
				Notes:              ex.Notes,
				Sets:               sets,
			})
		}

		if len(hevyExercises) == 0 {
			skipped++
			continue
		}

		// Estimate end time from duration
		endTime := w.Date.Add(parseDuration(w.Duration))

		req := HevyWorkoutRequest{
			Workout: HevyWorkoutBody{
				Title:       w.Name,
				Description: w.Notes,
				IsPrivate:   true,
				StartTime:   w.Date.Format(time.RFC3339),
				EndTime:     endTime.Format(time.RFC3339),
				Exercises:   hevyExercises,
			},
		}

		if *dryRun {
			fmt.Printf("[%d/%d] DRY RUN: %s (%s) - %d exercises\n",
				i+1, len(workouts), w.Name, w.Date.Format("2006-01-02"), len(hevyExercises))
			imported++
			continue
		}

		fmt.Printf("[%d/%d] Importing: %s (%s) - %d exercises... ",
			i+1, len(workouts), w.Name, w.Date.Format("2006-01-02"), len(hevyExercises))

		err := createWorkout(req)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			skipped++
			continue
		}
		fmt.Println("OK")
		imported++

		// Rate limit
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Printf("\nDone! Imported: %d, Skipped: %d\n", imported, skipped)
}

func fetchAllTemplates() ([]HevyExerciseTemplate, error) {
	var all []HevyExerciseTemplate
	page := 1
	for {
		url := fmt.Sprintf("%s/exercise_templates?page=%d&pageSize=10", baseURL, page)
		body, err := doGet(url)
		if err != nil {
			return nil, err
		}
		var resp HevyTemplatesResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.ExerciseTemplates...)
		if page >= resp.PageCount {
			break
		}
		page++
		time.Sleep(100 * time.Millisecond)
	}
	return all, nil
}

func parseCSV(path string) ([]Workout, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Build column index
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	// Verify required columns
	required := []string{"Date", "Workout Name", "Exercise Name", "Set Order", "Weight", "Reps"}
	for _, r := range required {
		if _, ok := colIdx[r]; !ok {
			return nil, fmt.Errorf("missing required column: %s", r)
		}
	}

	// Parse rows, group by (date, workout name)
	type workoutKey struct {
		date string
		name string
	}

	workoutMap := make(map[workoutKey]*Workout)
	var workoutOrder []workoutKey

	// Track exercise order within each workout
	exerciseOrder := make(map[workoutKey][]string)
	exerciseMap := make(map[workoutKey]map[string]*Exercise)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		dateStr := getCol(row, colIdx, "Date")
		wName := strings.TrimSpace(getCol(row, colIdx, "Workout Name"))
		exName := strings.TrimSpace(getCol(row, colIdx, "Exercise Name"))
		setOrder := strings.TrimSpace(getCol(row, colIdx, "Set Order"))
		weightStr := getCol(row, colIdx, "Weight")
		repsStr := getCol(row, colIdx, "Reps")
		notes := getCol(row, colIdx, "Notes")
		wNotes := getCol(row, colIdx, "Workout Notes")
		rpeStr := getCol(row, colIdx, "RPE")
		secondsStr := getCol(row, colIdx, "Seconds")

		// Skip "Rest Timer" rows
		if setOrder == "Rest Timer" {
			continue
		}

		key := workoutKey{date: dateStr, name: wName}

		if _, exists := workoutMap[key]; !exists {
			date, _ := time.Parse("2006-01-02 15:04:05", dateStr)
			dur := getCol(row, colIdx, "Duration")
			workoutMap[key] = &Workout{
				Name:     wName,
				Date:     date,
				Duration: dur,
				Notes:    strings.TrimSpace(wNotes),
			}
			workoutOrder = append(workoutOrder, key)
			exerciseMap[key] = make(map[string]*Exercise)
		}

		if _, exists := exerciseMap[key][exName]; !exists {
			exerciseMap[key][exName] = &Exercise{
				Name:  exName,
				Notes: strings.TrimSpace(notes),
			}
			exerciseOrder[key] = append(exerciseOrder[key], exName)
		}

		weight := parseFloat(weightStr)
		reps := parseInt(repsStr)
		rpe := parseFloat(rpeStr)
		seconds := parseInt(secondsStr)

		exerciseMap[key][exName].Sets = append(exerciseMap[key][exName].Sets, Set{
			Weight:  weight,
			Reps:    reps,
			RPE:     rpe,
			Seconds: seconds,
		})
	}

	// Assemble in order
	var workouts []Workout
	for _, key := range workoutOrder {
		w := workoutMap[key]
		for _, exName := range exerciseOrder[key] {
			w.Exercises = append(w.Exercises, *exerciseMap[key][exName])
		}
		workouts = append(workouts, *w)
	}

	return workouts, nil
}

func createWorkout(req HevyWorkoutRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", baseURL+"/workouts", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("api-key", apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func doGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func getCol(row []string, colIdx map[string]int, name string) string {
	idx, ok := colIdx[name]
	if !ok || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.Replace(s, ",", ".", 1)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseInt(s string) int {
	f := parseFloat(s)
	return int(f)
}

func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "m")
	minutes, err := strconv.Atoi(s)
	if err != nil {
		return 60 * time.Minute // default 1 hour
	}
	return time.Duration(minutes) * time.Minute
}
