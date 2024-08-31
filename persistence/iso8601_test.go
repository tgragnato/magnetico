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
