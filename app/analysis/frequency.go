package analysis

import (
	"sort"
	"time"

	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

// AnalyzeFrequency computes training frequency metrics.
func AnalyzeFrequency(workouts []hevy.Workout, templates []hevy.ExerciseTemplate, weeks int) *FrequencyReport {
	if len(workouts) == 0 || weeks <= 0 {
		return &FrequencyReport{MuscleFrequency: map[string]float64{}}
	}

	templateMap := buildTemplateMap(templates)
	cutoff := time.Now().AddDate(0, 0, -weeks*7)

	var filtered []hevy.Workout
	for _, w := range workouts {
		if !w.StartTime.Before(cutoff) {
			filtered = append(filtered, w)
		}
	}

	if len(filtered) == 0 {
		return &FrequencyReport{MuscleFrequency: map[string]float64{}}
	}

	// Workouts per week
	workoutsPerWeek := float64(len(filtered)) / float64(weeks)

	// Average session duration
	totalMinutes := 0.0
	for _, w := range filtered {
		dur := w.EndTime.Sub(w.StartTime).Minutes()
		if dur > 0 && dur < 300 { // sanity: skip >5h sessions
			totalMinutes += dur
		}
	}
	avgSessionMinutes := 0.0
	if len(filtered) > 0 {
		avgSessionMinutes = totalMinutes / float64(len(filtered))
	}

	// Muscle frequency: count distinct workout days per muscle / weeks
	muscleDays := map[string]map[string]bool{} // muscle -> set of date strings
	for _, w := range filtered {
		dateKey := w.StartTime.Format("2006-01-02")
		for _, ex := range w.Exercises {
			tmpl, ok := templateMap[ex.ExerciseTemplateID]
			if !ok {
				continue
			}
			muscle := tmpl.PrimaryMuscleGroup
			if muscle == "" {
				continue
			}
			if muscleDays[muscle] == nil {
				muscleDays[muscle] = map[string]bool{}
			}
			muscleDays[muscle][dateKey] = true
		}
	}

	muscleFrequency := map[string]float64{}
	for muscle, days := range muscleDays {
		muscleFrequency[muscle] = float64(len(days)) / float64(weeks)
	}

	// Average rest days between workouts
	avgRestDays := 0.0
	if len(filtered) > 1 {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].StartTime.Before(filtered[j].StartTime)
		})

		// Collect unique workout dates
		dates := []time.Time{}
		seen := map[string]bool{}
		for _, w := range filtered {
			dk := w.StartTime.Format("2006-01-02")
			if !seen[dk] {
				seen[dk] = true
				dates = append(dates, w.StartTime)
			}
		}

		if len(dates) > 1 {
			totalGap := 0.0
			for i := 1; i < len(dates); i++ {
				gap := dates[i].Sub(dates[i-1]).Hours() / 24.0
				totalGap += gap
			}
			avgRestDays = totalGap/float64(len(dates)-1) - 1 // subtract 1 for the workout day itself
			if avgRestDays < 0 {
				avgRestDays = 0
			}
		}
	}

	return &FrequencyReport{
		WorkoutsPerWeek:   workoutsPerWeek,
		AvgSessionMinutes: avgSessionMinutes,
		MuscleFrequency:   muscleFrequency,
		AvgRestDays:       avgRestDays,
	}
}
