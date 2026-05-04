package model

import (
	"strconv"
	"strings"
)

const (
	TKGrade           = -1
	KindergartenGrade = 0
)

func ParseGrade(grade string) (int, error) {
	normalized := strings.TrimSpace(strings.ToUpper(grade))

	switch normalized {
	case "TK":
		return TKGrade, nil
	case "K":
		return KindergartenGrade, nil
	case "4/5":
		return 4, nil
	default:
		return strconv.Atoi(normalized)
	}
}

func GradeLabel(grade int) string {
	switch grade {
	case TKGrade:
		return "TK"
	case KindergartenGrade:
		return "K"
	default:
		return strconv.Itoa(grade)
	}
}
