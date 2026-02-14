package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

// TemplateResponse is the JSON structure returned by the LLM for workout templates.
type TemplateResponse struct {
	Title     string             `json:"title"`
	Exercises []TemplateExercise `json:"exercises"`
}

// TemplateExercise is a single exercise in a generated template.
type TemplateExercise struct {
	ExerciseTemplateID string        `json:"exercise_template_id"`
	Title              string        `json:"title"`
	Sets               []TemplateSet `json:"sets"`
}

// TemplateSet is a single set in a generated template.
type TemplateSet struct {
	Type     string   `json:"type"`
	Reps     int      `json:"reps"`
	WeightKG float64  `json:"weight_kg"`
	RPE      *float64 `json:"rpe,omitempty"`
}

// ParseTemplateJSON extracts and parses JSON from an LLM response.
// Handles responses wrapped in markdown code blocks.
func ParseTemplateJSON(raw string) (*TemplateResponse, error) {
	raw = strings.TrimSpace(raw)

	// Strip markdown code blocks if present
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		// Remove first line (```json or ```) and last line (```)
		start := 1
		end := len(lines) - 1
		if end > start {
			raw = strings.Join(lines[start:end], "\n")
		}
	}

	raw = strings.TrimSpace(raw)

	var resp TemplateResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	if resp.Title == "" {
		return nil, fmt.Errorf("template has no title")
	}
	if len(resp.Exercises) == 0 {
		return nil, fmt.Errorf("template has no exercises")
	}

	return &resp, nil
}

// ValidateAndFixTemplate checks that all exercise IDs exist in the cache.
// If an ID is invalid but the title matches a known template, the ID is auto-corrected.
func ValidateAndFixTemplate(t *TemplateResponse, templates []hevy.ExerciseTemplate) error {
	idSet := make(map[string]bool)
	titleToID := make(map[string]string) // lowercase title -> ID
	for _, tmpl := range templates {
		idSet[tmpl.ID] = true
		titleToID[strings.ToLower(tmpl.Title)] = tmpl.ID
	}

	for i, ex := range t.Exercises {
		if idSet[ex.ExerciseTemplateID] {
			continue
		}
		// ID is invalid — try to fix by matching title
		if fixedID, ok := titleToID[strings.ToLower(ex.Title)]; ok {
			t.Exercises[i].ExerciseTemplateID = fixedID
		} else {
			return fmt.Errorf("unknown exercise: %s (ID %s not found, title not matched)", ex.Title, ex.ExerciseTemplateID)
		}
	}
	return nil
}

// FormatTemplatePreview renders a template as Telegram-friendly text.
func FormatTemplatePreview(t *TemplateResponse) string {
	var b strings.Builder
	b.WriteString(t.Title + "\n")

	for _, ex := range t.Exercises {
		b.WriteString(fmt.Sprintf("\n  %s\n", ex.Title))
		for _, s := range ex.Sets {
			line := fmt.Sprintf("    %s:", s.Type)
			if s.WeightKG > 0 {
				line += fmt.Sprintf(" %.0fkg x %d", s.WeightKG, s.Reps)
			} else if s.Reps > 0 {
				line += fmt.Sprintf(" %d reps", s.Reps)
			}
			if s.RPE != nil && *s.RPE > 0 {
				line += fmt.Sprintf(" @RPE %.0f", *s.RPE)
			}
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

// ToCreateRoutineRequest converts a template to a Hevy CreateRoutineRequest.
func ToCreateRoutineRequest(t *TemplateResponse) hevy.CreateRoutineRequest {
	exercises := make([]hevy.RoutineExercise, len(t.Exercises))
	for i, ex := range t.Exercises {
		sets := make([]hevy.RoutineSet, len(ex.Sets))
		for j, s := range ex.Sets {
			set := hevy.RoutineSet{
				Index: j,
				Type:  s.Type,
			}
			if s.Reps > 0 {
				reps := s.Reps
				set.Reps = &reps
			}
			if s.WeightKG > 0 {
				w := s.WeightKG
				set.WeightKG = &w
			}
			sets[j] = set
		}

		exercises[i] = hevy.RoutineExercise{
			Index:              i,
			Title:              ex.Title,
			ExerciseTemplateID: ex.ExerciseTemplateID,
			Sets:               sets,
		}
	}

	return hevy.CreateRoutineRequest{
		Routine: hevy.CreateRoutineBody{
			Title:     t.Title,
			Exercises: exercises,
		},
	}
}
