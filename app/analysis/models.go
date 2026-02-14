package analysis

import "time"

type AnalysisResult struct {
	Period    string
	StartDate time.Time
	EndDate   time.Time
	Weeks     int
	Volume    *VolumeReport
	Overload  *OverloadReport
	Balance   *BalanceReport
	Frequency *FrequencyReport
}

type VolumeReport struct {
	Muscles []MuscleVolume
}

type MuscleVolume struct {
	Muscle        string
	WeeklySets    float64 // primary: 1.0 credit, secondary: 0.5
	WeeklyTonnage float64 // sum(weight * reps) / weeks
	Status        string  // "low", "optimal", "high"
}

type OverloadReport struct {
	Exercises []ExerciseProgress
}

type ExerciseProgress struct {
	ExerciseName string
	TemplateID   string
	Sessions     []SessionE1RM
	Trend        string // "progressing", "stalling", "regressing"
	BestE1RM     float64
	LatestE1RM   float64
}

type SessionE1RM struct {
	Date time.Time
	E1RM float64
}

type BalanceReport struct {
	PushSets   float64
	PullSets   float64
	LegsSets   float64
	PushPull   float64  // ratio
	UpperLower float64  // ratio
	Flags      []string // imbalance warnings
}

type FrequencyReport struct {
	WorkoutsPerWeek   float64
	AvgSessionMinutes float64
	MuscleFrequency   map[string]float64 // sessions/week per muscle group
	AvgRestDays       float64
}
