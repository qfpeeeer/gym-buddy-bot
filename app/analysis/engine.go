package analysis

import (
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/qfpeeeer/gym-buddy-bot/app/hevy"
)

const defaultWeeks = 4

// Analyze runs all sub-analyzers and returns a combined result.
func Analyze(workouts []hevy.Workout, templates []hevy.ExerciseTemplate, weeks int) *AnalysisResult {
	if weeks <= 0 {
		weeks = defaultWeeks
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -weeks*7)

	return &AnalysisResult{
		Period:    fmt.Sprintf("Last %d weeks", weeks),
		StartDate: startDate,
		EndDate:   endDate,
		Weeks:     weeks,
		Volume:    AnalyzeVolume(workouts, templates, weeks),
		Overload:  AnalyzeOverload(workouts, templates),
		Balance:   AnalyzeBalance(workouts, templates, weeks),
		Frequency: AnalyzeFrequency(workouts, templates, weeks),
	}
}

// FormatVolumeReport formats volume analysis for Telegram.
func FormatVolumeReport(r *VolumeReport) string {
	if r == nil || len(r.Muscles) == 0 {
		return "No volume data available."
	}

	var b strings.Builder
	b.WriteString("<b>Weekly Volume</b>\n\n")

	for _, m := range r.Muscles {
		icon := ""
		switch m.Status {
		case "low":
			icon = "🟡"
		case "optimal":
			icon = "🟢"
		case "high":
			icon = "🔴"
		}
		b.WriteString(fmt.Sprintf("%s %s: %.1f sets/wk",
			icon, html.EscapeString(formatMuscle(m.Muscle)), m.WeeklySets))
		if m.WeeklyTonnage > 0 {
			b.WriteString(fmt.Sprintf(" (%.0f kg)", m.WeeklyTonnage))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n🟡 &lt;10 sets · 🟢 10-20 · 🔴 &gt;20")
	return b.String()
}

// FormatOverloadReport formats progressive overload analysis for Telegram.
func FormatOverloadReport(r *OverloadReport) string {
	if r == nil || len(r.Exercises) == 0 {
		return "No progressive overload data available."
	}

	var b strings.Builder
	b.WriteString("<b>Progressive Overload</b>\n\n")

	progressing := []ExerciseProgress{}
	stalling := []ExerciseProgress{}
	regressing := []ExerciseProgress{}

	for _, ex := range r.Exercises {
		switch ex.Trend {
		case "progressing":
			progressing = append(progressing, ex)
		case "stalling":
			stalling = append(stalling, ex)
		case "regressing":
			regressing = append(regressing, ex)
		}
	}

	if len(progressing) > 0 {
		b.WriteString("📈 <b>Progressing</b>\n")
		for _, ex := range progressing {
			b.WriteString(fmt.Sprintf("  %s: e1RM %.1f kg (best: %.1f)\n",
				html.EscapeString(ex.ExerciseName), ex.LatestE1RM, ex.BestE1RM))
		}
		b.WriteString("\n")
	}

	if len(stalling) > 0 {
		b.WriteString("➡️ <b>Stalling</b>\n")
		for _, ex := range stalling {
			b.WriteString(fmt.Sprintf("  %s: e1RM %.1f kg (best: %.1f)\n",
				html.EscapeString(ex.ExerciseName), ex.LatestE1RM, ex.BestE1RM))
		}
		b.WriteString("\n")
	}

	if len(regressing) > 0 {
		b.WriteString("📉 <b>Regressing</b>\n")
		for _, ex := range regressing {
			b.WriteString(fmt.Sprintf("  %s: e1RM %.1f kg (best: %.1f)\n",
				html.EscapeString(ex.ExerciseName), ex.LatestE1RM, ex.BestE1RM))
		}
	}

	return b.String()
}

// FormatOverloadSummary formats a condensed overload summary for Telegram (names only, no e1RM).
func FormatOverloadSummary(r *OverloadReport) string {
	if r == nil || len(r.Exercises) == 0 {
		return "No progressive overload data available."
	}

	var progressing, stalling, regressing []string

	for _, ex := range r.Exercises {
		name := html.EscapeString(ex.ExerciseName)
		switch ex.Trend {
		case "progressing":
			progressing = append(progressing, name)
		case "stalling":
			stalling = append(stalling, name)
		case "regressing":
			regressing = append(regressing, name)
		}
	}

	var b strings.Builder
	b.WriteString("<b>Progressive Overload</b>\n\n")

	if len(progressing) > 0 {
		b.WriteString(fmt.Sprintf("📈 <b>Progressing (%d):</b> %s\n", len(progressing), strings.Join(progressing, ", ")))
	}
	if len(stalling) > 0 {
		b.WriteString(fmt.Sprintf("➡️ <b>Stalling (%d):</b> %s\n", len(stalling), strings.Join(stalling, ", ")))
	}
	if len(regressing) > 0 {
		b.WriteString(fmt.Sprintf("📉 <b>Regressing (%d):</b> %s\n", len(regressing), strings.Join(regressing, ", ")))
	}

	return b.String()
}

// FormatBalanceReport formats balance analysis for Telegram.
func FormatBalanceReport(r *BalanceReport) string {
	if r == nil {
		return "No balance data available."
	}

	var b strings.Builder
	b.WriteString("<b>Muscle Balance</b>\n\n")

	b.WriteString(fmt.Sprintf("Push: %.1f sets/wk\n", r.PushSets))
	b.WriteString(fmt.Sprintf("Pull: %.1f sets/wk\n", r.PullSets))
	b.WriteString(fmt.Sprintf("Legs: %.1f sets/wk\n", r.LegsSets))
	b.WriteString("\n")

	if r.PushPull > 0 {
		b.WriteString(fmt.Sprintf("Push:Pull = %.2f:1\n", r.PushPull))
	}
	if r.UpperLower > 0 {
		b.WriteString(fmt.Sprintf("Upper:Lower = %.2f:1\n", r.UpperLower))
	}

	if len(r.Flags) > 0 {
		b.WriteString("\n⚠️ <b>Warnings:</b>\n")
		for _, f := range r.Flags {
			b.WriteString(fmt.Sprintf("  %s\n", f))
		}
	} else {
		b.WriteString("\n✅ Balance looks good!")
	}

	return b.String()
}

// FormatFrequencyReport formats frequency analysis for Telegram.
func FormatFrequencyReport(r *FrequencyReport) string {
	if r == nil {
		return "No frequency data available."
	}

	var b strings.Builder
	b.WriteString("<b>Training Frequency</b>\n\n")

	b.WriteString(fmt.Sprintf("Workouts/week: %.1f\n", r.WorkoutsPerWeek))
	b.WriteString(fmt.Sprintf("Avg session: %.0f min\n", r.AvgSessionMinutes))
	b.WriteString(fmt.Sprintf("Avg rest: %.1f days\n", r.AvgRestDays))

	if len(r.MuscleFrequency) > 0 {
		b.WriteString("\n<b>Muscle frequency</b> (sessions/wk):\n")

		type mf struct {
			muscle string
			freq   float64
		}
		var sorted []mf
		for m, f := range r.MuscleFrequency {
			sorted = append(sorted, mf{m, f})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].freq > sorted[j].freq
		})

		for _, m := range sorted {
			b.WriteString(fmt.Sprintf("  %s: %.1f\n", html.EscapeString(formatMuscle(m.muscle)), m.freq))
		}
	}

	return b.String()
}

// FormatFullReport formats all analysis results for Telegram.
func FormatFullReport(r *AnalysisResult) string {
	if r == nil {
		return "No analysis data available."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<b>Full Report</b> (%s)\n\n", r.Period))

	b.WriteString(FormatFrequencyReport(r.Frequency))
	b.WriteString("\n\n")
	b.WriteString(FormatVolumeReport(r.Volume))
	b.WriteString("\n\n")
	b.WriteString(FormatBalanceReport(r.Balance))
	b.WriteString("\n\n")
	b.WriteString(FormatOverloadReport(r.Overload))

	return b.String()
}

func formatMuscle(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}
