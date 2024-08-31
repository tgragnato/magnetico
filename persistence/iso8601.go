package persistence

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var (
	yearRE  = regexp.MustCompile(`^(\d{4})$`)
	monthRE = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
	weekRE  = regexp.MustCompile(`^(\d{4})-W(\d{2})$`)
	dayRE   = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	hourRE  = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})T(\d{2})$`)
)

type Granularity int

const (
	Year Granularity = iota
	Month
	Week
	Day
	Hour
)

// ParseISO8601 is **not** a function to parse all and every kind of valid ISO 8601
// date, nor it's intended to be, since we don't need that.
func ParseISO8601(s string) (*time.Time, Granularity, error) {
	if matches := yearRE.FindStringSubmatch(s); len(matches) != 0 {
		year := parseYear(matches[1])
		t := time.Date(year, time.December, daysOfMonth(time.December, year), 23, 59, 59, 0, time.UTC)
		return &t, Year, nil
	}

	if matches := monthRE.FindStringSubmatch(s); len(matches) != 0 {
		month := parseMonth(matches[2])
		year := parseYear(matches[1])
		t := time.Date(year, month, 31, 23, 59, 59, 0, time.UTC)
		return &t, Month, nil
	}

	if matches := weekRE.FindStringSubmatch(s); len(matches) != 0 {
		week := parseWeek(matches[2])
		year := parseYear(matches[1])
		t := time.Date(year, time.January, week*7, 23, 59, 59, 0, time.UTC)
		return &t, Week, nil
	}

	if matches := dayRE.FindStringSubmatch(s); len(matches) != 0 {
		month := parseMonth(matches[2])
		year := parseYear(matches[1])
		day := parseDay(matches[3], daysOfMonth(month, year))
		t := time.Date(year, month, day, 23, 59, 59, 0, time.UTC)
		return &t, Day, nil
	}

	if matches := hourRE.FindStringSubmatch(s); len(matches) != 0 {
		month := parseMonth(matches[2])
		year := parseYear(matches[1])
		hour := parseHour(matches[4])
		day := parseDay(matches[3], daysOfMonth(month, year))
		t := time.Date(year, month, day, hour, 59, 59, 0, time.UTC)
		return &t, Hour, nil
	}

	return nil, -1, fmt.Errorf("string does not match any formats")
}

func daysOfMonth(month time.Month, year int) int {
	switch month {
	case time.January:
		return 31
	case time.February:
		if isLeap(year) {
			return 29
		} else {
			return 28
		}
	case time.March:
		return 31
	case time.April:
		return 30
	case time.May:
		return 31
	case time.June:
		return 30
	case time.July:
		return 31
	case time.August:
		return 31
	case time.September:
		return 30
	case time.October:
		return 31
	case time.November:
		return 30
	case time.December:
		return 31
	default:
		return 0
	}
}

func isLeap(year int) bool {
	if year%4 != 0 {
		return false
	} else if year%100 != 0 {
		return true
	} else if year%400 != 0 {
		return false
	} else {
		return true
	}
}

func parseYear(s string) int {
	year, err := strconv.Atoi(s)
	if err != nil || year <= 1583 {
		return 0
	}
	return year
}

func parseMonth(s string) time.Month {
	month, err := strconv.Atoi(s)
	if err != nil || month <= 0 || month >= 13 {
		return time.Month(-1)
	}
	return time.Month(month)
}

func parseWeek(s string) int {
	week, err := strconv.Atoi(s)
	if err != nil || week <= 0 || week >= 54 {
		return -1
	}
	return week
}

func parseDay(s string, max int) int {
	day, err := strconv.Atoi(s)
	if err != nil || day <= 0 || day > max {
		return -1
	}
	return day
}

func parseHour(s string) int {
	hour, err := strconv.Atoi(s)
	if err != nil || hour <= -1 || hour >= 25 {
		return -1
	}
	return hour
}
