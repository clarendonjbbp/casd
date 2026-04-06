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
