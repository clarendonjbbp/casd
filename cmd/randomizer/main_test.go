package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitStudentNamesCommaSeparated(t *testing.T) {
	assert.Equal(t,
		[]string{"Alice Smith", "Bob Jones", "Cara Brown"},
		splitStudentNames("Alice Smith, Bob Jones, Cara Brown"),
	)
}

func TestSplitStudentNamesSemicolonSeparated(t *testing.T) {
	assert.Equal(t,
		[]string{"Alice Smith", "Bob Jones", "Cara Brown"},
		splitStudentNames("Alice Smith; Bob Jones; Cara Brown"),
	)
}

func TestSplitStudentNamesNewlineSeparated(t *testing.T) {
	assert.Equal(t,
		[]string{"Alice Smith", "Bob Jones", "Cara Brown"},
		splitStudentNames("Alice Smith\nBob Jones\nCara Brown"),
	)
}

func TestSplitStudentNamesNormalizesTabsAndLeadingAnd(t *testing.T) {
	assert.Equal(t,
		[]string{"Alice Smith", "Bob Jones", "Cara Brown"},
		splitStudentNames("Alice\tSmith, Bob Jones, and Cara Brown"),
	)
}

func TestUniqueTeacherReplacementDoesNotReuseNames(t *testing.T) {
	used := map[string]struct{}{
		"Emma": {},
		"Liam": {},
	}

	first := uniqueTeacherReplacement(used)
	used[first] = struct{}{}
	second := uniqueTeacherReplacement(used)

	assert.NotEqual(t, first, second)
	assert.NotEqual(t, "Emma", first)
	assert.NotEqual(t, "Liam", first)
	assert.NotEqual(t, "Emma", second)
	assert.NotEqual(t, "Liam", second)
}
