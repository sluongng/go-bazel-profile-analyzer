package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzer(t *testing.T) {
	expected := `start time: 2022-02-07 00:07:50 +0100 CET
launch: 1.385s 63.858197%
init: 269.153ms 12.409838%
target pattern evaluation: 245.444ms 11.316687%
interleaved loading-and-analysis: 177.022ms 8.161954%
preparation: 14.248ms 0.656933%
execution: 44.936ms 2.071864%
finish: 33.065ms 1.524528%
Total: 2.168868s 100%
`
	analysis, err := Analyze("bazel-profile.json")
	assert.Nil(t, err)

	assert.Equal(t, analysis.Summary.String(), expected)
	assert.Equal(t, analysis.BuildMetadata.BuildID, "478e7ca7-c4f6-4255-aef9-ae71b0f2abf2")
	assert.Equal(t, 270, len(analysis.TefData.Events()))
	assert.Equal(t, 3, len(analysis.TefData.Metadata()))
	assert.Equal(t, 1, len(analysis.CriticalPathComponents))
}
