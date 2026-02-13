package strong

import (
	"testing"
	"time"
)

func TestParseText_FullWorkout(t *testing.T) {
	input := `Upper B (chest dominated)
Thursday, 12 February 2026 at 18:39

Iso-Lateral Chest Press (Machine)
Set 1: 20 kg × 12
Set 2: 30 kg × 12
Set 3: 35 kg × 8

Incline Bench Press (Barbell)
Set 1: 40 kg × 12
Set 2: 60 kg × 8
Set 3: 70 kg × 1

Cable Crossover
Set 1: 25 kg × 12
Set 2: 25 kg × 10
Set 3: 25 kg × 12

Seated Row (Cable)
Set 1: 40 kg × 12
Set 2: 45 kg × 12
Set 3: 55 kg × 8

Lat Pulldown (Cable)
Set 1: 50 kg × 12
Set 2: 60 kg × 8
Set 3: 65 kg × 6

Notes: V handle

Triceps Extension (Cable)
Set 1: 20 kg × 10
Set 2: 17,5 kg × 10
Set 3: 17,5 kg × 8

Triceps Isolated Cable Pull (One Hand)
Set 1: 7,5 kg × 8
Set 2: 7,5 kg × 8
Set 3: 7,5 kg × 8
https://link.strong.app/uyndgrjx`

	workout, err := ParseText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if workout.Name != "Upper B (chest dominated)" {
		t.Errorf("expected workout name %q, got %q", "Upper B (chest dominated)", workout.Name)
	}

	expectedDate := time.Date(2026, 2, 12, 18, 39, 0, 0, time.UTC)
	if !workout.Date.Equal(expectedDate) {
		t.Errorf("expected date %v, got %v", expectedDate, workout.Date)
	}

	if len(workout.Exercises) != 7 {
		t.Fatalf("expected 7 exercises, got %d", len(workout.Exercises))
	}

	// Check first exercise
	ex := workout.Exercises[0]
	if ex.Name != "Iso-Lateral Chest Press (Machine)" {
		t.Errorf("exercise 0: expected name %q, got %q", "Iso-Lateral Chest Press (Machine)", ex.Name)
	}
	if len(ex.Sets) != 3 {
		t.Fatalf("exercise 0: expected 3 sets, got %d", len(ex.Sets))
	}
	if ex.Sets[0].Weight != 20 || ex.Sets[0].Reps != 12 {
		t.Errorf("exercise 0 set 0: expected 20kg×12, got %.1fkg×%d", ex.Sets[0].Weight, ex.Sets[0].Reps)
	}
	if ex.Sets[2].Weight != 35 || ex.Sets[2].Reps != 8 {
		t.Errorf("exercise 0 set 2: expected 35kg×8, got %.1fkg×%d", ex.Sets[2].Weight, ex.Sets[2].Reps)
	}

	// Check decimal weight (European comma format: 17,5)
	tricepsExt := workout.Exercises[5]
	if tricepsExt.Name != "Triceps Extension (Cable)" {
		t.Errorf("exercise 5: expected name %q, got %q", "Triceps Extension (Cable)", tricepsExt.Name)
	}
	if tricepsExt.Sets[1].Weight != 17.5 {
		t.Errorf("exercise 5 set 1: expected weight 17.5, got %.1f", tricepsExt.Sets[1].Weight)
	}

	// Check notes on Lat Pulldown
	latPulldown := workout.Exercises[4]
	if latPulldown.Name != "Lat Pulldown (Cable)" {
		t.Errorf("exercise 4: expected name %q, got %q", "Lat Pulldown (Cable)", latPulldown.Name)
	}
	if latPulldown.Notes != "V handle" {
		t.Errorf("exercise 4: expected notes %q, got %q", "V handle", latPulldown.Notes)
	}

	// Check last exercise with decimal comma weights
	lastEx := workout.Exercises[6]
	if lastEx.Name != "Triceps Isolated Cable Pull (One Hand)" {
		t.Errorf("exercise 6: expected name %q, got %q", "Triceps Isolated Cable Pull (One Hand)", lastEx.Name)
	}
	if lastEx.Sets[0].Weight != 7.5 {
		t.Errorf("exercise 6 set 0: expected weight 7.5, got %.1f", lastEx.Sets[0].Weight)
	}
}

func TestParseText_BodyweightExercise(t *testing.T) {
	input := `Morning Workout
Monday, 3 February 2026 at 07:00

Push Ups
Set 1: 20
Set 2: 15
Set 3: 12`

	workout, err := ParseText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workout.Exercises) != 1 {
		t.Fatalf("expected 1 exercise, got %d", len(workout.Exercises))
	}

	ex := workout.Exercises[0]
	if ex.Sets[0].Weight != 0 {
		t.Errorf("expected 0 weight for bodyweight, got %.1f", ex.Sets[0].Weight)
	}
	if ex.Sets[0].Reps != 20 {
		t.Errorf("expected 20 reps, got %d", ex.Sets[0].Reps)
	}
}

func TestParseText_TooShort(t *testing.T) {
	_, err := ParseText("just one line")
	if err == nil {
		t.Error("expected error for text too short")
	}
}

func TestParseText_BadDate(t *testing.T) {
	input := `Workout
not a real date

Push Ups
Set 1: 10`

	_, err := ParseText(input)
	if err == nil {
		t.Error("expected error for bad date")
	}
}

func TestParseText_NoExercises(t *testing.T) {
	input := `Workout
Monday, 3 February 2026 at 07:00
`

	_, err := ParseText(input)
	if err == nil {
		t.Error("expected error for no exercises")
	}
}

func TestParseText_ASCIIMultiplicationSign(t *testing.T) {
	input := `Workout
Monday, 3 February 2026 at 07:00

Bench Press
Set 1: 60 kg x 10`

	workout, err := ParseText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if workout.Exercises[0].Sets[0].Weight != 60 || workout.Exercises[0].Sets[0].Reps != 10 {
		t.Errorf("expected 60kg×10, got %.1fkg×%d",
			workout.Exercises[0].Sets[0].Weight, workout.Exercises[0].Sets[0].Reps)
	}
}

func TestParseText_WindowsLineEndings(t *testing.T) {
	input := "Workout\r\nMonday, 3 February 2026 at 07:00\r\n\r\nSquat\r\nSet 1: 100 kg × 5\r\n"

	workout, err := ParseText(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if workout.Exercises[0].Sets[0].Weight != 100 {
		t.Errorf("expected 100kg, got %.1f", workout.Exercises[0].Sets[0].Weight)
	}
}
