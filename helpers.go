package main

import (
	"strconv"
	"strings"
	"time"
)

type BySchedule []string

// parseDuration parses durations of days, weeks and months (in the most simplistic way)
// since time.ParseDuration only supports up to hours https://github.com/golang/go/issues/11473
// If there's a parsing error, return 0 and the error.  Originally, this returned MAXINT64 and the
// error, but time.ParseDuration(foo) returns 0 on error and I wanted to stay consistent.
func parseDuration(d string) (time.Duration, error) {
	switch {
	case strings.HasSuffix(d, "d"):
		t := strings.TrimSuffix(d, "d")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*24) * time.Hour, nil
	case strings.HasSuffix(d, "w"):
		t := strings.TrimSuffix(d, "w")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*7*24) * time.Hour, nil
	case strings.HasSuffix(d, "mo"):
		t := strings.TrimSuffix(d, "mo")
		num, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Duration(num*30*24) * time.Hour, nil
	default:
		return time.ParseDuration(d)
	}
}

// Len is required to satisfy sort.Interface
func (s BySchedule) Len() int {
	return len(s)
}

// Swap is required to satisfy sort.Interface
func (s BySchedule) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less is required to satisfy sort.Interface
func (s BySchedule) Less(i, j int) bool {
	di, _ := parseDuration(s[i])
	dj, _ := parseDuration(s[j])
	return di < dj
}
