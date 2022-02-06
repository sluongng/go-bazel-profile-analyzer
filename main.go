package analyzer

import (
	"fmt"
	"os"
	"time"

	"github.com/omaskery/teffy/pkg/events"
	tefio "github.com/omaskery/teffy/pkg/io"
)

// Reverse mapping of ProfilePhase.java
// Reference: https://github.com/bazelbuild/bazel@b2a943434d413be2e25fbc9cd57a4f84afd6d4c5/-/blob/src/main/java/com/google/devtools/build/lib/profiler/ProfilePhase.java
var NameProfilePhases = map[string]string{
	"Launch Blaze":                  "launch",
	"Initialize command":            "init",
	"Evaluate target patterns":      "target pattern evaluation",
	"Load and analyze dependencies": "interleaved loading-and-analysis",
	"Analyze licenses":              "license checking",
	"Prepare for build":             "preparation",
	"Build artifacts":               "execution",
	"Complete build":                "finish",
	"unknown":                       "unknown",
}

// StatsSummary holds the duration of each phases inside the profile
type StatsSummary struct {
	Launch                  *time.Duration
	Init                    *time.Duration
	TargetPatternEvaluation *time.Duration
	LoadingAnalysis         *time.Duration
	LicenseChecking         *time.Duration
	Preparation             *time.Duration
	Execution               *time.Duration
	Finish                  *time.Duration
	Unknown                 *time.Duration
	Total                   time.Duration
}

// AddValue help add phases values to StatSummary
// only known phases listedn in analyzer.NameProfilePhases are accepted
func (ss *StatsSummary) AddValue(phaseName string, val time.Duration) {
	switch phaseName {
	case "launch":
		ss.Launch = &val
	case "init":
		ss.Init = &val
	case "target pattern evaluation":
		ss.TargetPatternEvaluation = &val
	case "interleaved loading-and-analysis":
		ss.LoadingAnalysis = &val
	case "license checking":
		ss.LicenseChecking = &val
	case "preparation":
		ss.Preparation = &val
	case "execution":
		ss.Execution = &val
	case "finish":
		ss.Finish = &val
	case "unknown":
		ss.Unknown = &val
	default:
		return
	}

	ss.Total += val
}

func (ss *StatsSummary) String() string {
	out := "\n"
	if ss.Launch != nil {
		percentage := float64(*ss.Launch) / float64(ss.Total) * 100
		out += fmt.Sprintf("launch: %s %f%%\n", ss.Launch, percentage)
	}
	if ss.Init != nil {
		percentage := float64(*ss.Init) / float64(ss.Total) * 100
		out += fmt.Sprintf("init: %s %f%%\n", ss.Init, percentage)
	}
	if ss.TargetPatternEvaluation != nil {
		percentage := float64(*ss.TargetPatternEvaluation) / float64(ss.Total) * 100
		out += fmt.Sprintf("target pattern evaluation: %s %f%%\n", ss.TargetPatternEvaluation, percentage)
	}
	if ss.LoadingAnalysis != nil {
		percentage := float64(*ss.LoadingAnalysis) / float64(ss.Total) * 100
		out += fmt.Sprintf("interleaved loading-and-analysis: %s %f%%\n", ss.LoadingAnalysis, percentage)
	}
	if ss.LicenseChecking != nil {
		percentage := float64(*ss.LicenseChecking) / float64(ss.Total) * 100
		out += fmt.Sprintf("license checking: %s %f%%\n", ss.LicenseChecking, percentage)
	}
	if ss.Preparation != nil {
		percentage := float64(*ss.Preparation) / float64(ss.Total) * 100
		out += fmt.Sprintf("preparation: %s %f%%\n", ss.Preparation, percentage)
	}
	if ss.Execution != nil {
		percentage := float64(*ss.Execution) / float64(ss.Total) * 100
		out += fmt.Sprintf("execution: %s %f%%\n", ss.Execution, percentage)
	}
	if ss.Finish != nil {
		percentage := float64(*ss.Finish) / float64(ss.Total) * 100
		out += fmt.Sprintf("finish: %s %f%%\n", ss.Finish, percentage)
	}
	if ss.Unknown != nil {
		percentage := float64(*ss.Unknown) / float64(ss.Total) * 100
		out += fmt.Sprintf("Unknown: %s %f%%\n", ss.Unknown, percentage)
	}

	return out + fmt.Sprintf("Total: %s 100%%\n", ss.Total)
}

type BazelProfileAnalysis struct {
	Summary *StatsSummary
	// TODO: add more stuffs
}

// Analyze helps analyze bazel JSON profile
// is the exact equivalent of `bazel analyze-profile`
func Analyze(profileFilePath string) (*BazelProfileAnalysis, error) {
	f, err := os.Open(profileFilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %v", err)
	}
	defer f.Close()

	traces, err := tefio.ParseJsonObj(f)
	if err != nil {
		return nil, fmt.Errorf("cannot parse file: %v", err)
	}

	// Mapping thread IDs and thread names
	// ID is unique but names can be duplicated
	// TODO: do we need/care about this?
	threadName := make(map[int64]string)

	phaseSummaryStats := &StatsSummary{}

	lastPhaseEvent := ""
	lastPhaseEventTimeStamp := 0 * time.Microsecond
	maxEndTime := 0 * time.Microsecond
	for _, event := range traces.Events() {
		eventTimeStamp := time.Duration(event.Core().Timestamp) * time.Microsecond

		if eventTimeStamp > maxEndTime {
			maxEndTime = eventTimeStamp
		}

		switch e := event.(type) {
		case *events.Complete:
			if isBuildPhaseMarker(e) {
				if e.Core().Name != "Launch Blaze" {
					// all build phase marker are of phase Instant
					// except for Launch Blaze
					return nil, fmt.Errorf("caught unexpected event %s", e.Core().Name)
				}

				if lastPhaseEvent != "" {
					phaseSummaryStats.AddValue(lastPhaseEvent, eventTimeStamp-lastPhaseEventTimeStamp)
				}

				lastPhaseEvent = NameProfilePhases[e.Core().Name]
				lastPhaseEventTimeStamp = eventTimeStamp
				continue
			}

			if isCricitalPathComponent(e) {
				// TODO: handle critical events
				continue
			}

			// TODO: handle normal Complete events
		case *events.Instant:
			if !isBuildPhaseMarker(e) {
				// We are not interested in non-build-phase-marker events
				// for now.
				continue
			}

			if lastPhaseEvent != "" {
				phaseSummaryStats.AddValue(lastPhaseEvent, eventTimeStamp-lastPhaseEventTimeStamp)
			}

			lastPhaseEvent = NameProfilePhases[e.Core().Name]
			lastPhaseEventTimeStamp = eventTimeStamp
		case *events.MetadataThreadName:
			threadName[*e.Core().ThreadID] = e.ThreadName
		case *events.Counter:
			// TODO: not sure what to do with action counters and cpu counters
			// ignore them for now
		default:
		}
	}

	if lastPhaseEvent != "" {
		phaseSummaryStats.AddValue(lastPhaseEvent, maxEndTime-lastPhaseEventTimeStamp)
	}

	return &BazelProfileAnalysis{
		phaseSummaryStats,
	}, nil
}

func isBuildPhaseMarker(e events.Event) bool {
	for _, cat := range e.Core().Categories {
		if cat == "build phase marker" {
			return true
		}
	}

	return false
}

func isCricitalPathComponent(e *events.Complete) bool {
	for _, cat := range e.Core().Categories {
		if cat == "critical path component" {
			return true
		}
	}

	return false
}
