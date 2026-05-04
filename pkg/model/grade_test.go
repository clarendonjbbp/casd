package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGrade(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{input: "TK", expected: TKGrade},
		{input: "tk", expected: TKGrade},
		{input: "K", expected: KindergartenGrade},
		{input: "4/5", expected: 4},
		{input: "2", expected: 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			grade, err := ParseGrade(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, grade)
		})
	}
}

func TestGradeLabel(t *testing.T) {
	assert.Equal(t, "TK", GradeLabel(TKGrade))
	assert.Equal(t, "K", GradeLabel(KindergartenGrade))
	assert.Equal(t, "3", GradeLabel(3))
}
