package persistence

import (
	"testing"
	"time"
)

var validDates = []struct {
	date        string
	granularity Granularity
}{
	{
		"2018",
		Year,
	},
	{
		"2018-04",
		Month,
	},
	{
		"2018-W16",
		Week,
	},
	{
		"2018-04-20",
		Day,
	},
	{
		"2018-04-20T15",
		Hour,
	},
}

func TestParseISO8601(t *testing.T) {
	t.Parallel()

	for i, date := range validDates {
		_, gr, err := ParseISO8601(date.date)
		if err != nil {
			t.Errorf("Error while parsing valid date #%d: %s", i+1, err.Error())
			continue
		}

		if gr != date.granularity {
			t.Errorf("Granularity of the date #%d is wrong! Got %d (expected %d)",
				i+1, gr, date.granularity)
			continue
		}
	}
}

func TestParseISO8601MonthBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		date string
		want time.Time
	}{
		{"2026-02", time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)},
		{"2024-02", time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			got, gran, err := ParseISO8601(tt.date)
			if err != nil {
				t.Fatalf("ParseISO8601(%q): %v", tt.date, err)
			}
			if gran != Month {
				t.Fatalf("ParseISO8601(%q) granularity = %v, want %v", tt.date, gran, Month)
			}
			if !got.Equal(tt.want) {
				t.Errorf("ParseISO8601(%q) = %v, want %v", tt.date, got, tt.want)
			}
		})
	}
}

func TestStatisticsToTimeMonthBoundary(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	got := statisticsToTime(from, Month, 1)
	want := time.Date(2026, time.March, 31, 23, 59, 59, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("statisticsToTime(%v, Month, 1) = %v, want %v", from, got, want)
	}
}

func TestDaysOfMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		month time.Month
		year  int
		want  int
	}{
		{time.January, 2022, 31},
		{time.February, 2022, 28},
		{time.February, 2024, 29},
		{time.March, 2022, 31},
		{time.April, 2022, 30},
		{time.May, 2022, 31},
		{time.June, 2022, 30},
		{time.July, 2022, 31},
		{time.August, 2022, 31},
		{time.September, 2022, 30},
		{time.October, 2022, 31},
		{time.November, 2022, 30},
		{time.December, 2022, 31},
	}

	for _, tt := range tests {
		got := daysOfMonth(tt.month, tt.year)
		if got != tt.want {
			t.Errorf("daysOfMonth(%v, %v) = %v, want %v", tt.month, tt.year, got, tt.want)
		}
	}
}

func TestIsLeap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		year int
		want bool
	}{
		{2000, true},
		{2004, true},
		{2100, false},
		{2200, false},
		{2400, true},
	}

	for _, tt := range tests {
		got := isLeap(tt.year)
		if got != tt.want {
			t.Errorf("isLeap(%v) = %v, want %v", tt.year, got, tt.want)
		}
	}
}
