package analysis

import (
	"math"
	"sort"
	"time"

	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

type sessionData struct {
	date time.Time
	e1rm float64
}

// AnalyzeOverload tracks progressive overload via estimated 1RM trends.
func AnalyzeOverload(workouts []hevy.Workout, templates []hevy.ExerciseTemplate) *OverloadReport {
	if len(workouts) == 0 {
		return &OverloadReport{}
	}

	templateMap := buildTemplateMap(templates)

	// Group best e1RM per exercise per workout session
	exerciseSessions := map[string][]sessionData{} // templateID -> sessions
	exerciseNames := map[string]string{}           // templateID -> name

	for _, w := range workouts {
		for _, ex := range w.Exercises {
			tmpl, ok := templateMap[ex.ExerciseTemplateID]
			if !ok {
				continue
			}
			// Only weight_reps exercises
			if tmpl.Type != "weight_reps" {
				continue
			}

			bestE1RM := 0.0
			for _, s := range ex.Sets {
				if s.Type == "warmup" {
					continue
				}
				if s.WeightKG == nil || s.Reps == nil {
					continue
				}
				weight := *s.WeightKG
				reps := *s.Reps
				if weight <= 0 || reps <= 0 {
					continue
				}
				// Epley formula: e1RM = weight * (1 + reps/30)
				e1rm := weight * (1 + float64(reps)/30.0)
				if e1rm > bestE1RM {
					bestE1RM = e1rm
				}
			}

			if bestE1RM > 0 {
				exerciseSessions[ex.ExerciseTemplateID] = append(
					exerciseSessions[ex.ExerciseTemplateID],
					sessionData{date: w.StartTime, e1rm: bestE1RM},
				)
				exerciseNames[ex.ExerciseTemplateID] = ex.Title
			}
		}
	}

	var exercises []ExerciseProgress
	for templateID, sessions := range exerciseSessions {
		if len(sessions) < 2 {
			continue
		}

		// Sort by date
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].date.Before(sessions[j].date)
		})

		// Build session e1RM list
		sessionE1RMs := make([]SessionE1RM, len(sessions))
		bestE1RM := 0.0
		for i, s := range sessions {
			sessionE1RMs[i] = SessionE1RM{Date: s.date, E1RM: s.e1rm}
			if s.e1rm > bestE1RM {
				bestE1RM = s.e1rm
			}
		}

		latestE1RM := sessions[len(sessions)-1].e1rm

		// Trend: compare avg of last 3 vs previous 3
		trend := computeTrend(sessions)

		exercises = append(exercises, ExerciseProgress{
			ExerciseName: exerciseNames[templateID],
			TemplateID:   templateID,
			Sessions:     sessionE1RMs,
			Trend:        trend,
			BestE1RM:     math.Round(bestE1RM*10) / 10,
			LatestE1RM:   math.Round(latestE1RM*10) / 10,
		})
	}

	// Sort by exercise name
	sort.Slice(exercises, func(i, j int) bool {
		return exercises[i].ExerciseName < exercises[j].ExerciseName
	})

	return &OverloadReport{Exercises: exercises}
}

func computeTrend(sessions []sessionData) string {
	n := len(sessions)
	if n < 3 {
		// With only 2 sessions, compare directly
		if sessions[n-1].e1rm > sessions[0].e1rm*1.025 {
			return "progressing"
		} else if sessions[n-1].e1rm < sessions[0].e1rm*0.975 {
			return "regressing"
		}
		return "stalling"
	}

	// Average of last 3 vs previous 3 (or all previous if < 6 sessions)
	recentCount := 3
	if recentCount > n/2 {
		recentCount = n / 2
	}

	recentAvg := 0.0
	for i := n - recentCount; i < n; i++ {
		recentAvg += sessions[i].e1rm
	}
	recentAvg /= float64(recentCount)

	previousCount := recentCount
	if previousCount > n-recentCount {
		previousCount = n - recentCount
	}
	previousAvg := 0.0
	for i := n - recentCount - previousCount; i < n-recentCount; i++ {
		previousAvg += sessions[i].e1rm
	}
	previousAvg /= float64(previousCount)

	if previousAvg == 0 {
		return "stalling"
	}

	change := (recentAvg - previousAvg) / previousAvg
	if change > 0.025 {
		return "progressing"
	} else if change < -0.025 {
		return "regressing"
	}
	return "stalling"
}
