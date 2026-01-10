package handlers

import (
	"testing"
	"time"
)

func TestDiff(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Kolkata")
	tests := []struct {
		name      string
		a         time.Time
		b         time.Time
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{
			name:      "One day difference",
			a:         time.Date(2023, 1, 1, 0, 0, 0, 0, loc),
			b:         time.Date(2023, 1, 2, 0, 0, 0, 0, loc),
			wantYear:  0,
			wantMonth: 0,
			wantDay:   1,
		},
		{
			name:      "One month difference",
			a:         time.Date(2023, 1, 1, 0, 0, 0, 0, loc),
			b:         time.Date(2023, 2, 1, 0, 0, 0, 0, loc),
			wantYear:  0,
			wantMonth: 1,
			wantDay:   0,
		},
		{
			name:      "Negative day borrow (Jan to Feb)",
			a:         time.Date(2023, 1, 30, 0, 0, 0, 0, loc),
			b:         time.Date(2023, 2, 1, 0, 0, 0, 0, loc),
			wantYear:  0,
			wantMonth: 0,
			wantDay:   2,
		},
		{
			name:      "Negative day borrow (Feb to Mar)",
			a:         time.Date(2023, 2, 28, 0, 0, 0, 0, loc),
			b:         time.Date(2023, 3, 1, 0, 0, 0, 0, loc),
			wantYear:  0,
			wantMonth: 0,
			wantDay:   1,
		},
		{
			name:      "Negative day borrow (Leap year)",
			a:         time.Date(2024, 2, 28, 0, 0, 0, 0, loc),
			b:         time.Date(2024, 3, 1, 0, 0, 0, 0, loc),
			wantYear:  0,
			wantMonth: 0,
			wantDay:   2,
		},
		{
			name:      "Full year difference",
			a:         time.Date(2023, 1, 1, 0, 0, 0, 0, loc),
			b:         time.Date(2024, 1, 1, 0, 0, 0, 0, loc),
			wantYear:  1,
			wantMonth: 0,
			wantDay:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYear, gotMonth, gotDay := diff(tt.a, tt.b)
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("diff() = (%v, %v, %v), want (%v, %v, %v)", gotYear, gotMonth, gotDay, tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

func TestSetAgeInYears(t *testing.T) {
	tests := []struct {
		y, m, d int
		want    string
	}{
		{1, 0, 0, "1 year"},
		{2, 1, 1, "2 years 1 month 1 day"},
		{0, 2, 0, "2 months"},
		{0, 0, 5, "5 days"},
		{1, 0, 3, "1 year 3 days"},
		{0, 0, 0, "0 days"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := setAgeInYears(tt.y, tt.m, tt.d); got != tt.want {
				t.Errorf("setAgeInYears() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgeInDays(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Kolkata")
	c := &Controller{location: loc}

	tests := []struct {
		name      string
		eventDate time.Time
		currTime  time.Time
		wantDays  int
	}{
		{
			name:      "Same day",
			eventDate: time.Date(2023, 1, 1, 23, 0, 0, 0, loc),
			currTime:  time.Date(2023, 1, 1, 10, 0, 0, 0, loc),
			wantDays:  0,
		},
		{
			name:      "Next day morning, less than 24h",
			eventDate: time.Date(2023, 1, 1, 23, 0, 0, 0, loc),
			currTime:  time.Date(2023, 1, 2, 1, 0, 0, 0, loc),
			wantDays:  1,
		},
		{
			name:      "Next day evening, more than 24h",
			eventDate: time.Date(2023, 1, 1, 23, 0, 0, 0, loc),
			currTime:  time.Date(2023, 1, 2, 23, 5, 0, 0, loc),
			wantDays:  1,
		},
		{
			name: "Previous day evening, event in UTC vs Local",
			// Event in UTC: 2023-01-01 22:00:00 UTC = 2023-01-02 03:30:00 IST
			eventDate: time.Date(2023, 1, 1, 22, 0, 0, 0, time.UTC),
			currTime:  time.Date(2023, 1, 2, 10, 0, 0, 0, loc),
			wantDays:  0, // Both are Jan 2nd in IST
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventDate := tt.eventDate.In(c.location)
			currTime := tt.currTime.In(c.location)

			d1 := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, c.location)
			d2 := time.Date(currTime.Year(), currTime.Month(), currTime.Day(), 0, 0, 0, 0, c.location)
			gotDays := int(d2.Sub(d1).Hours() / 24)

			if gotDays != tt.wantDays {
				t.Errorf("%s: AgeInDays = %v, want %v", tt.name, gotDays, tt.wantDays)
			}
		})
	}
}
