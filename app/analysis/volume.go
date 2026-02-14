package analysis

import (
	"sort"
	"time"

	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

// AnalyzeVolume computes weekly set volume and tonnage per muscle group.
func AnalyzeVolume(workouts []hevy.Workout, templates []hevy.ExerciseTemplate, weeks int) *VolumeReport {
	if len(workouts) == 0 || weeks <= 0 {
		return &VolumeReport{}
	}

	templateMap := buildTemplateMap(templates)
	cutoff := time.Now().AddDate(0, 0, -weeks*7)

	// Accumulate total sets and tonnage per muscle
	muscleSets := map[string]float64{}
	muscleTonnage := map[string]float64{}

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

				// Set credit: primary 1.0, secondary 0.5
				if tmpl.PrimaryMuscleGroup != "" {
					muscleSets[tmpl.PrimaryMuscleGroup] += 1.0
				}
				for _, sec := range tmpl.SecondaryMuscleGroups {
					muscleSets[sec] += 0.5
				}

				// Tonnage
				weight := 0.0
				reps := 0
				if s.WeightKG != nil {
					weight = *s.WeightKG
				}
				if s.Reps != nil {
					reps = *s.Reps
				}
				tonnage := weight * float64(reps)
				if tonnage > 0 {
					if tmpl.PrimaryMuscleGroup != "" {
						muscleTonnage[tmpl.PrimaryMuscleGroup] += tonnage
					}
					for _, sec := range tmpl.SecondaryMuscleGroups {
						muscleTonnage[sec] += tonnage * 0.5
					}
				}
			}
		}
	}

	// Convert to weekly averages
	var muscles []MuscleVolume
	for muscle, sets := range muscleSets {
		weeklySets := sets / float64(weeks)
		weeklyTonnage := muscleTonnage[muscle] / float64(weeks)

		status := "optimal"
		if weeklySets < 10 {
			status = "low"
		} else if weeklySets > 20 {
			status = "high"
		}

		muscles = append(muscles, MuscleVolume{
			Muscle:        muscle,
			WeeklySets:    weeklySets,
			WeeklyTonnage: weeklyTonnage,
			Status:        status,
		})
	}

	sort.Slice(muscles, func(i, j int) bool {
		return muscles[i].WeeklySets > muscles[j].WeeklySets
	})

	return &VolumeReport{Muscles: muscles}
}

func buildTemplateMap(templates []hevy.ExerciseTemplate) map[string]hevy.ExerciseTemplate {
	m := make(map[string]hevy.ExerciseTemplate, len(templates))
	for _, t := range templates {
		m[t.ID] = t
	}
	return m
}
