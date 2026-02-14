package analysis

import (
	"fmt"
	"time"

	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

var pushMuscles = map[string]bool{
	"chest": true, "shoulders": true, "triceps": true,
}

var pullMuscles = map[string]bool{
	"lats": true, "upper_back": true, "traps": true,
	"biceps": true, "forearms": true,
}

var legMuscles = map[string]bool{
	"quadriceps": true, "hamstrings": true, "glutes": true,
	"calves": true, "abductors": true, "adductors": true,
}

// AnalyzeBalance computes push/pull/legs ratios and flags imbalances.
func AnalyzeBalance(workouts []hevy.Workout, templates []hevy.ExerciseTemplate, weeks int) *BalanceReport {
	if len(workouts) == 0 || weeks <= 0 {
		return &BalanceReport{}
	}

	templateMap := buildTemplateMap(templates)
	cutoff := time.Now().AddDate(0, 0, -weeks*7)

	pushSets := 0.0
	pullSets := 0.0
	legsSets := 0.0

	for _, w := range workouts {
		if w.StartTime.Before(cutoff) {
			continue
		}

		for _, ex := range w.Exercises {
			tmpl, ok := templateMap[ex.ExerciseTemplateID]
			if !ok {
				continue
			}

			for _, s := range ex.Sets {
				if s.Type == "warmup" {
					continue
				}

				muscle := tmpl.PrimaryMuscleGroup
				if pushMuscles[muscle] {
					pushSets++
				} else if pullMuscles[muscle] {
					pullSets++
				} else if legMuscles[muscle] {
					legsSets++
				}
			}
		}
	}

	// Weekly averages
	pushSets /= float64(weeks)
	pullSets /= float64(weeks)
	legsSets /= float64(weeks)

	pushPull := 0.0
	if pullSets > 0 {
		pushPull = pushSets / pullSets
	}

	upperSets := pushSets + pullSets
	upperLower := 0.0
	if legsSets > 0 {
		upperLower = upperSets / legsSets
	}

	var flags []string
	if pushPull > 1.3 {
		flags = append(flags, fmt.Sprintf("Push:Pull ratio is %.1f:1 (high) — consider adding more pull work", pushPull))
	} else if pushPull < 0.7 && pushPull > 0 {
		flags = append(flags, fmt.Sprintf("Push:Pull ratio is %.1f:1 (low) — consider adding more push work", pushPull))
	}
	if upperLower > 1.5 {
		flags = append(flags, fmt.Sprintf("Upper:Lower ratio is %.1f:1 (high) — consider adding more leg work", upperLower))
	} else if upperLower < 0.8 && upperLower > 0 {
		flags = append(flags, fmt.Sprintf("Upper:Lower ratio is %.1f:1 (low) — consider adding more upper body work", upperLower))
	}

	return &BalanceReport{
		PushSets:   pushSets,
		PullSets:   pullSets,
		LegsSets:   legsSets,
		PushPull:   pushPull,
		UpperLower: upperLower,
		Flags:      flags,
	}
}
