package strong

import (
	"strings"
	"testing"
)

func TestParseCSV_SingleWorkout(t *testing.T) {
	csv := `Date,Workout Name,Duration,Exercise Name,Set Order,Weight,Reps,Distance,Seconds,Notes,Workout Number
2026-02-12 18:39:00,Upper B (chest dominated),1h 5m,Iso-Lateral Chest Press (Machine),1,20,12,0,0,,1
2026-02-12 18:39:00,Upper B (chest dominated),1h 5m,Iso-Lateral Chest Press (Machine),2,30,12,0,0,,1
2026-02-12 18:39:00,Upper B (chest dominated),1h 5m,Iso-Lateral Chest Press (Machine),3,35,8,0,0,,1
2026-02-12 18:39:00,Upper B (chest dominated),1h 5m,Incline Bench Press (Barbell),1,40,12,0,0,,1
2026-02-12 18:39:00,Upper B (chest dominated),1h 5m,Incline Bench Press (Barbell),2,60,8,0,0,,1`

	workouts, err := ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workouts) != 1 {
		t.Fatalf("expected 1 workout, got %d", len(workouts))
	}

	w := workouts[0]
	if w.Name != "Upper B (chest dominated)" {
		t.Errorf("expected workout name %q, got %q", "Upper B (chest dominated)", w.Name)
	}

	if len(w.Exercises) != 2 {
		t.Fatalf("expected 2 exercises, got %d", len(w.Exercises))
	}

	if len(w.Exercises[0].Sets) != 3 {
		t.Errorf("expected 3 sets for exercise 0, got %d", len(w.Exercises[0].Sets))
	}

	if w.Exercises[0].Sets[0].Weight != 20 || w.Exercises[0].Sets[0].Reps != 12 {
		t.Errorf("exercise 0 set 0: expected 20kg×12, got %.1fkg×%d",
			w.Exercises[0].Sets[0].Weight, w.Exercises[0].Sets[0].Reps)
	}

	if len(w.Exercises[1].Sets) != 2 {
		t.Errorf("expected 2 sets for exercise 1, got %d", len(w.Exercises[1].Sets))
	}
}

func TestParseCSV_MultipleWorkouts(t *testing.T) {
	csv := `Date,Workout Name,Duration,Exercise Name,Set Order,Weight,Reps,Distance,Seconds,Notes,Workout Number
2026-02-10 10:00:00,Leg Day,,Squat,1,100,5,0,0,,1
2026-02-10 10:00:00,Leg Day,,Squat,2,120,3,0,0,,1
2026-02-12 18:39:00,Upper B,,Bench Press,1,60,10,0,0,,2
2026-02-12 18:39:00,Upper B,,Bench Press,2,80,6,0,0,,2`

	workouts, err := ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workouts) != 2 {
		t.Fatalf("expected 2 workouts, got %d", len(workouts))
	}

	if workouts[0].Name != "Leg Day" {
		t.Errorf("expected first workout %q, got %q", "Leg Day", workouts[0].Name)
	}
	if workouts[1].Name != "Upper B" {
		t.Errorf("expected second workout %q, got %q", "Upper B", workouts[1].Name)
	}
}

func TestParseCSV_WithNotes(t *testing.T) {
	csv := `Date,Workout Name,Duration,Exercise Name,Set Order,Weight,Reps,Distance,Seconds,Notes,Workout Number
2026-02-12 18:39:00,Workout,,Lat Pulldown,1,50,12,0,0,V handle,1
2026-02-12 18:39:00,Workout,,Lat Pulldown,2,60,8,0,0,,1`

	workouts, err := ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if workouts[0].Exercises[0].Notes != "V handle" {
		t.Errorf("expected notes %q, got %q", "V handle", workouts[0].Exercises[0].Notes)
	}
}

func TestParseCSV_MissingRequiredColumn(t *testing.T) {
	csv := `Date,Workout Name,Exercise Name,Weight,Reps
2026-02-12 18:39:00,Workout,Bench Press,60,10`

	_, err := ParseCSV(strings.NewReader(csv))
	if err == nil {
		t.Error("expected error for missing Set Order column")
	}
}

func TestParseCSV_EmptyCSV(t *testing.T) {
	csv := `Date,Workout Name,Duration,Exercise Name,Set Order,Weight,Reps,Distance,Seconds,Notes,Workout Number`

	workouts, err := ParseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workouts) != 0 {
		t.Errorf("expected 0 workouts, got %d", len(workouts))
	}
}
