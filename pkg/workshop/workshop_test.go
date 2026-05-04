package workshop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/clarendonjbbp/casd/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWorkshopsSupportsTKGradeRange(t *testing.T) {
	workshopsCSV := `Workshop,Grades,S1,S2,S3,S4,Capacity,Room
A1 - TK Garden,TK-K,y,n,n,n,12,101
`
	path := filepath.Join(t.TempDir(), "artworkshops.csv")
	require.NoError(t, os.WriteFile(path, []byte(workshopsCSV), 0o644))

	workshops, err := ReadWorkshops(path, model.ArtWorkshop)
	require.NoError(t, err)

	workshop := workshops["A1"]
	require.NotNil(t, workshop)
	assert.Equal(t, model.TKGrade, workshop.MinGrade)
	assert.Equal(t, model.KindergartenGrade, workshop.MaxGrade)
	assert.True(t, workshop.WithinGradeRange(model.TKGrade))
	assert.True(t, workshop.WithinGradeRange(model.KindergartenGrade))
}
