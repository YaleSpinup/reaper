package main

import (
	"sort"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := map[string]time.Duration{
		"1s":  time.Duration(1) * time.Second,
		"2m":  time.Duration(2) * time.Minute,
		"3h":  time.Duration(3) * time.Hour,
		"5d":  time.Duration(5*24) * time.Hour,
		"2w":  time.Duration(2*7*24) * time.Hour,
		"2mo": time.Duration(2*30*24) * time.Hour,
	}

	for s, d := range tests {
		actual, err := parseDuration(s)
		if err != nil {
			t.Errorf("Failed to parse duration %s", err.Error())
		}

		if d != actual {
			t.Errorf("Expected time string %s to be duration %v, got %v", s, d, actual)
		}
	}

	bad, err := parseDuration("someotherstring")
	if err == nil {
		t.Errorf("Expected error from bad string to not be nil, got duration %v", bad)
	}
}

func TestBySchedule(t *testing.T) {
	l := []string{"1w", "2d", "10s", "3s", "1mo", "27d"}
	expected := []string{"3s", "10s", "2d", "1w", "27d", "1mo"}

	sort.Sort(BySchedule(l))

	if len(l) != len(expected) {
		t.Errorf("Expected lengths to be equal after sort.  Expected: %d, got: %d", len(expected), len(l))

	}

	for i, v := range expected {
		if v != l[i] {
			t.Errorf("Expected %s, got %s", v, l[i])
		}
	}
}
