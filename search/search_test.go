package search

import (
	"reflect"
	"testing"
)

var testFilters = map[string]string{
	"lizard": "king",
	"fuzzy":  "baby ducks",
	"zombie": "dust",
	"space":  "coyote",
}

var testTermQuery = []TermQuery{
	TermQuery{Term: "lizard", Value: "king"},
	TermQuery{Term: "fuzzy", Value: "baby ducks"},
	TermQuery{Term: "zombie", Value: "dust"},
	TermQuery{Term: "space", Value: "coyote"},
}

func TestNewTermQuery(t *testing.T) {
	actual := NewTermQueryList(testFilters)
	if len(actual) != len(testTermQuery) {
		t.Errorf("Expected generated term query list and test term query list to have the same length (%d/%d)", len(actual), len(testTermQuery))
	}

LOOP:
	for _, q := range testTermQuery {
		t.Logf("Checking test query: %+v", q)

		for _, a := range actual {
			t.Logf("Comparing against actual value %+v", a)
			if reflect.DeepEqual(q, a) {
				t.Logf("Got a match")
				continue LOOP
			}
		}

		t.Errorf("Expected generated term query %+v list to contain %+v", actual, q)
	}
}

func TestFinder(t *testing.T) {
	t.Log("No tests")
}
