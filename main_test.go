package analyzer

import (
	"strings"
	"testing"
)

var expected = `launch: 1.385s 63.858197%
init: 269.153ms 12.409838%
target pattern evaluation: 245.444ms 11.316687%
interleaved loading-and-analysis: 177.022ms 8.161954%
preparation: 14.248ms 0.656933%
execution: 44.936ms 2.071864%
finish: 33.065ms 1.524528%
Total: 2.168868s 100%`

func TestAnalyzer(t *testing.T) {
	analysis, err := Analyze("bazel-profile.json")
	if err != nil {
		t.Errorf("could not analyze: %v\n", err)
	}

	got := strings.TrimSpace(analysis.Summary.String())
	if got != expected {
		t.Errorf("=== expected\n%s\n===got\n%v\n", expected, got)
	}
}
