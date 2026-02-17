package llm

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/qfpeeeer/gym-buddy-bot/app/analysis"
	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
	"github.com/qfpeeeer/gym-buddy-bot/app/storage"
)

// BuildContext assembles a compact training context string from local DB data.
func BuildContext(
	workouts []hevy.Workout,
	templates []hevy.ExerciseTemplate,
	prefs *storage.UserPreferences,
	result *analysis.AnalysisResult,
) string {
	var b strings.Builder

	buildProfile(&b, workouts, result)
	buildPreferences(&b, prefs)
	buildVolume(&b, result)
	buildOverload(&b, result)
	buildBalance(&b, result)
	buildRecentWorkouts(&b, workouts, 5)

	return b.String()
}

// BuildContextWithExercises includes available exercises for template generation.
func BuildContextWithExercises(
	workouts []hevy.Workout,
	templates []hevy.ExerciseTemplate,
	prefs *storage.UserPreferences,
	result *analysis.AnalysisResult,
) string {
	ctx := BuildContext(workouts, templates, prefs, result)

	var b strings.Builder
	b.WriteString(ctx)
	buildAvailableExercises(&b, workouts, templates)

	return b.String()
}

// BuildInsightsContext assembles a slim context for AI insights — key metrics only, no recent workouts.
func BuildInsightsContext(
	workouts []hevy.Workout,
	templates []hevy.ExerciseTemplate,
	prefs *storage.UserPreferences,
	result *analysis.AnalysisResult,
) string {
	var b strings.Builder

	buildProfile(&b, workouts, result)
	buildPreferences(&b, prefs)
	buildVolume(&b, result)
	buildOverloadInsights(&b, result)
	buildBalance(&b, result)

	return b.String()
}

func buildOverloadInsights(b *strings.Builder, result *analysis.AnalysisResult) {
	if result == nil || result.Overload == nil || len(result.Overload.Exercises) == 0 {
		return
	}

	var progressing, stalling, regressing []analysis.ExerciseProgress
	for _, ex := range result.Overload.Exercises {
		switch ex.Trend {
		case "progressing":
			progressing = append(progressing, ex)
		case "stalling":
			stalling = append(stalling, ex)
		case "regressing":
			regressing = append(regressing, ex)
		}
	}

	// Sort each group by magnitude of change (biggest movers first).
	sortByChange := func(exercises []analysis.ExerciseProgress) {
		sort.Slice(exercises, func(i, j int) bool {
			return math.Abs(overloadChangePercent(exercises[i])) > math.Abs(overloadChangePercent(exercises[j]))
		})
	}
	sortByChange(progressing)
	sortByChange(stalling)
	sortByChange(regressing)

	b.WriteString("=== PROGRESSIVE OVERLOAD ===\n")
	b.WriteString(fmt.Sprintf("Summary: %d progressing, %d stalling, %d regressing\n\n",
		len(progressing), len(stalling), len(regressing)))

	const cap = 5

	if len(progressing) > 0 {
		b.WriteString("Top progressing:\n")
		for i, ex := range progressing {
			if i >= cap {
				break
			}
			pct := overloadChangePercent(ex)
			b.WriteString(fmt.Sprintf("  %s: %.1f -> %.1f kg (%+.1f%%)\n",
				ex.ExerciseName, overloadFirstE1RM(ex), ex.LatestE1RM, pct))
		}
		b.WriteString("\n")
	}

	if len(stalling) > 0 {
		b.WriteString("Top stalling:\n")
		for i, ex := range stalling {
			if i >= cap {
				break
			}
			b.WriteString(fmt.Sprintf("  %s: %.1f kg (best: %.1f)\n",
				ex.ExerciseName, ex.LatestE1RM, ex.BestE1RM))
		}
		b.WriteString("\n")
	}

	if len(regressing) > 0 {
		b.WriteString("Top regressing:\n")
		for i, ex := range regressing {
			if i >= cap {
				break
			}
			pct := overloadChangePercent(ex)
			b.WriteString(fmt.Sprintf("  %s: %.1f -> %.1f kg (%+.1f%%)\n",
				ex.ExerciseName, overloadFirstE1RM(ex), ex.LatestE1RM, pct))
		}
		b.WriteString("\n")
	}
}

func overloadChangePercent(ex analysis.ExerciseProgress) float64 {
	first := overloadFirstE1RM(ex)
	if first <= 0 {
		return 0
	}
	return ((ex.LatestE1RM - first) / first) * 100
}

func overloadFirstE1RM(ex analysis.ExerciseProgress) float64 {
	if len(ex.Sessions) > 0 {
		return ex.Sessions[0].E1RM
	}
	return 0
}

func buildProfile(b *strings.Builder, workouts []hevy.Workout, result *analysis.AnalysisResult) {
	b.WriteString("=== TRAINING PROFILE ===\n")
	b.WriteString(fmt.Sprintf("Workouts logged: %d\n", len(workouts)))

	if len(workouts) >= 2 {
		first := workouts[0].StartTime
		last := workouts[len(workouts)-1].StartTime
		// workouts are ASC order from GetAllWorkouts
		if first.After(last) {
			first, last = last, first
		}
		months := int(last.Sub(first).Hours() / 24 / 30)
		b.WriteString(fmt.Sprintf("Period: %s — %s (%d months)\n",
			first.Format("Jan 2006"), last.Format("Jan 2006"), months))
	}

	if result != nil && result.Frequency != nil {
		freq := result.Frequency
		b.WriteString(fmt.Sprintf("Frequency: %.1f workouts/week\n", freq.WorkoutsPerWeek))
		b.WriteString(fmt.Sprintf("Avg session: %.0f min\n", freq.AvgSessionMinutes))
		b.WriteString(fmt.Sprintf("Avg rest days: %.1f\n", freq.AvgRestDays))
	}
	b.WriteString("\n")
}

func buildPreferences(b *strings.Builder, prefs *storage.UserPreferences) {
	if prefs == nil || (prefs.Goals == "" && prefs.Notes == "") {
		return
	}
	b.WriteString("=== USER PREFERENCES ===\n")
	if prefs.Goals != "" {
		b.WriteString(fmt.Sprintf("Goals: %s\n", prefs.Goals))
	}
	if prefs.Notes != "" {
		b.WriteString(fmt.Sprintf("Notes: %s\n", prefs.Notes))
	}
	b.WriteString("\n")
}

func buildVolume(b *strings.Builder, result *analysis.AnalysisResult) {
	if result == nil || result.Volume == nil || len(result.Volume.Muscles) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("=== VOLUME (last %d weeks) ===\n", result.Weeks))
	for _, m := range result.Volume.Muscles {
		b.WriteString(fmt.Sprintf("%s: %.1f sets/wk, %.0f kg tonnage [%s]\n",
			m.Muscle, m.WeeklySets, m.WeeklyTonnage, m.Status))
	}
	b.WriteString("\n")
}

func buildOverload(b *strings.Builder, result *analysis.AnalysisResult) {
	if result == nil || result.Overload == nil || len(result.Overload.Exercises) == 0 {
		return
	}
	b.WriteString("=== PROGRESSIVE OVERLOAD ===\n")
	for _, ex := range result.Overload.Exercises {
		trend := strings.ToUpper(ex.Trend)
		if ex.BestE1RM > 0 {
			pct := 0.0
			if len(ex.Sessions) >= 2 {
				first := ex.Sessions[0].E1RM
				if first > 0 {
					pct = ((ex.LatestE1RM - first) / first) * 100
				}
			}
			b.WriteString(fmt.Sprintf("%s: %s e1RM %.1f->%.1f kg (%+.1f%%)\n",
				ex.ExerciseName, trend, ex.Sessions[0].E1RM, ex.LatestE1RM, pct))
		}
	}
	b.WriteString("\n")
}

func buildBalance(b *strings.Builder, result *analysis.AnalysisResult) {
	if result == nil || result.Balance == nil {
		return
	}
	bal := result.Balance
	line := fmt.Sprintf("=== BALANCE ===\nPush:Pull = %.2f | Upper:Lower = %.2f",
		bal.PushPull, bal.UpperLower)
	if len(bal.Flags) > 0 {
		line += " [!" + strings.Join(bal.Flags, ", ") + "]"
	}
	b.WriteString(line + "\n\n")
}

func buildRecentWorkouts(b *strings.Builder, workouts []hevy.Workout, count int) {
	if len(workouts) == 0 {
		return
	}

	// workouts from GetAllWorkouts are ASC, we want most recent first
	recent := make([]hevy.Workout, len(workouts))
	copy(recent, workouts)
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].StartTime.After(recent[j].StartTime)
	})

	if len(recent) > count {
		recent = recent[:count]
	}

	b.WriteString("=== RECENT WORKOUTS ===\n")
	for _, w := range recent {
		duration := w.EndTime.Sub(w.StartTime)
		b.WriteString(fmt.Sprintf("%s \"%s\" (%d min)\n",
			w.StartTime.Format("2006-01-02"), w.Title, int(duration.Minutes())))

		for _, ex := range w.Exercises {
			b.WriteString(fmt.Sprintf("  %s:", ex.Title))
			setParts := formatSetGroups(ex.Sets)
			b.WriteString(" " + strings.Join(setParts, ", ") + "\n")
		}
	}
	b.WriteString("\n")
}

// formatSetGroups groups consecutive sets with same weight/reps for compact display.
func formatSetGroups(sets []hevy.WorkoutSet) []string {
	var parts []string
	for i := 0; i < len(sets); {
		s := sets[i]
		weight := 0.0
		reps := 0
		if s.WeightKG != nil {
			weight = *s.WeightKG
		}
		if s.Reps != nil {
			reps = *s.Reps
		}

		// Count consecutive identical sets
		count := 1
		for j := i + 1; j < len(sets); j++ {
			ns := sets[j]
			nw, nr := 0.0, 0
			if ns.WeightKG != nil {
				nw = *ns.WeightKG
			}
			if ns.Reps != nil {
				nr = *ns.Reps
			}
			if nw == weight && nr == reps && ns.Type == s.Type {
				count++
			} else {
				break
			}
		}

		var part string
		if weight > 0 && reps > 0 {
			part = fmt.Sprintf("%dx%d@%.0fkg", count, reps, weight)
		} else if reps > 0 {
			part = fmt.Sprintf("%dx%d", count, reps)
		} else if s.DurationSeconds != nil && *s.DurationSeconds > 0 {
			part = fmt.Sprintf("%dx%ds", count, *s.DurationSeconds)
		} else {
			i += count
			continue
		}

		if s.Type == "warmup" {
			part += "(w)"
		}

		parts = append(parts, part)
		i += count
	}
	return parts
}

func buildAvailableExercises(b *strings.Builder, workouts []hevy.Workout, templates []hevy.ExerciseTemplate) {
	// Build a set of exercise template IDs the user actually uses
	usedIDs := make(map[string]bool)
	for _, w := range workouts {
		for _, ex := range w.Exercises {
			usedIDs[ex.ExerciseTemplateID] = true
		}
	}

	// Build template lookup
	templateMap := make(map[string]hevy.ExerciseTemplate)
	for _, t := range templates {
		templateMap[t.ID] = t
	}

	// Collect used templates, sorted by title
	type entry struct {
		id, title, muscle, equipment string
	}
	var entries []entry
	for id := range usedIDs {
		t, ok := templateMap[id]
		if !ok {
			continue
		}
		entries = append(entries, entry{id: t.ID, title: t.Title, muscle: t.PrimaryMuscleGroup, equipment: t.Equipment})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].title < entries[j].title })

	// Find latest weight for each exercise
	latestWeight := make(map[string]float64)
	// workouts are ASC, so later entries overwrite earlier
	for _, w := range workouts {
		for _, ex := range w.Exercises {
			for _, s := range ex.Sets {
				if s.WeightKG != nil && *s.WeightKG > 0 {
					latestWeight[ex.ExerciseTemplateID] = *s.WeightKG
				}
			}
		}
	}

	b.WriteString("=== AVAILABLE EXERCISES ===\n")
	for _, e := range entries {
		line := fmt.Sprintf("ID: %s | %s | %s | %s", e.id, e.title, e.muscle, e.equipment)
		if w, ok := latestWeight[e.id]; ok {
			line += fmt.Sprintf(" | last: %.0fkg", w)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
}
