package causality

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var timezoneSuffix = regexp.MustCompile(`(Z|[+-][0-9]{2}:[0-9]{2})$`)

func EvaluateCases(cases []DurationCase) []CaseResult {
	results := make([]CaseResult, 0, len(cases))
	for _, item := range cases {
		results = append(results, evaluateCase(item))
	}
	return results
}

func SummaryFor(results []CaseResult) Summary {
	summary := Summary{Total: len(results)}
	for _, result := range results {
		switch result.Decision {
		case StateClosed:
			summary.Closed++
		case StateUnknown:
			summary.Unknown++
		case StateRefuted:
			summary.Refuted++
		}
	}
	return summary
}

func evaluateCase(item DurationCase) CaseResult {
	result := CaseResult{
		CaseID:           item.CaseID,
		Category:         item.Category,
		ExpectedDecision: item.ExpectedDecision,
		ExpectedReason:   item.ExpectedReason,
		Decision:         StateUnknown,
		Reason:           "CASE_NOT_EVALUATED",
		Resolution:       "UNKNOWN",
		Description:      item.Description,
		Violations:       []string{},
	}
	if item.IndependentObservations {
		return evaluateIndependentObservations(item, result)
	}
	start, ok := findObservation(item.Observations, item.StartObservationID)
	if !ok {
		return unknownResult(result, UnknownTuple{
			Stage:         "DURATION",
			Step:          "READ_START_TIME",
			Reason:        "START_OBSERVATION_MISSING",
			UnknownClass:  "DIRECT_MISSING",
			NextOperation: "RESTORE_START_OBSERVATION",
			BlockedBy:     []string{"start-observation"},
		})
	}
	end, ok := findObservation(item.Observations, item.EndObservationID)
	if !ok {
		return unknownResult(result, UnknownTuple{
			Stage:         "DURATION",
			Step:          "READ_END_TIME",
			Reason:        "END_OBSERVATION_MISSING",
			UnknownClass:  "DIRECT_MISSING",
			NextOperation: "RESTORE_END_OBSERVATION",
			BlockedBy:     []string{"end-observation"},
		})
	}
	return evaluatePair(item, result, start, end)
}

func evaluateIndependentObservations(item DurationCase, result CaseResult) CaseResult {
	if len(item.Observations) == 0 {
		return unknownResult(result, UnknownTuple{
			Stage:         "DURATION",
			Step:          "READ_OBSERVATIONS",
			Reason:        "OBSERVATIONS_MISSING",
			UnknownClass:  "DIRECT_MISSING",
			NextOperation: "RESTORE_OBSERVATIONS",
			BlockedBy:     []string{"observations"},
		})
	}
	for _, observation := range item.Observations {
		start, err := parseEndpoint(observation, true)
		if err != nil {
			return classifyEndpointError(result, err)
		}
		end, err := parseEndpoint(observation, false)
		if err != nil {
			return classifyEndpointError(result, err)
		}
		if end.Before(start) {
			return refutedResult(result, "REFUTED_CLOCK_ORDER", []string{"NEGATIVE_DURATION"})
		}
		duration := end.Sub(start).Milliseconds()
		result.DerivedObservations = append(result.DerivedObservations, DerivedObservation{
			ObservationID: observation.ObservationID,
			OperationID:   observation.OperationID,
			Scope:         observation.Scope,
			DurationMS:    duration,
		})
	}
	result.Decision = StateClosed
	result.Resolution = "EXACT"
	result.Reason = "SEPARATE_OBSERVATIONS_NOT_AGGREGATED"
	result.AggregateForbidden = true
	return result
}

func evaluatePair(item DurationCase, result CaseResult, start Observation, end Observation) CaseResult {
	if start.OperationID != end.OperationID {
		return refutedResult(result, "OPERATION_ID_MISMATCH", []string{"OPERATION_ID_MISMATCH"})
	}
	violations := make([]string, 0, 3)
	if start.RunID != end.RunID {
		violations = append(violations, "RUN_ID_MISMATCH")
	}
	if start.JobID != end.JobID {
		violations = append(violations, "JOB_ID_MISMATCH")
	}
	if start.Provider != end.Provider {
		violations = append(violations, "PROVIDER_MISMATCH")
	}
	if len(violations) > 0 {
		return refutedResult(result, "CROSS_RUN_JOB_PROVIDER_SUBTRACTION_FORBIDDEN", violations)
	}
	if start.ClockDomain == "" || end.ClockDomain == "" || start.ClockDomain != end.ClockDomain {
		if start.ClockDomain != end.ClockDomain && start.ClockDomain != "" && end.ClockDomain != "" {
			return refutedResult(result, "CLOCK_DOMAIN_MISMATCH", []string{"CLOCK_DOMAIN_MISMATCH"})
		}
		return unknownResult(result, UnknownTuple{
			Stage:         "DURATION",
			Step:          "VALIDATE_CLOCK_DOMAIN",
			Reason:        "CLOCK_DOMAIN_UNKNOWN",
			UnknownClass:  "DIRECT_UNKNOWN",
			NextOperation: "DECLARE_CLOCK_DOMAIN",
			BlockedBy:     []string{"clock-domain"},
		})
	}
	if start.ArtifactID == "" || end.ArtifactID == "" {
		return unknownResult(result, UnknownTuple{
			Stage:         "EVIDENCE",
			Step:          "READ_ARTIFACT",
			Reason:        "ARTIFACT_MISSING",
			UnknownClass:  "DIRECT_MISSING",
			NextOperation: "RESTORE_ARTIFACT",
			BlockedBy:     []string{"artifact"},
		})
	}
	startTime, err := parseEndpoint(start, true)
	if err != nil {
		return classifyEndpointError(result, err)
	}
	endTime, err := parseEndpoint(end, false)
	if err != nil {
		return classifyEndpointError(result, err)
	}
	duration := endTime.Sub(startTime).Milliseconds()
	if duration < 0 {
		if item.ClampToZero {
			return refutedResult(result, "CLAMP_TO_ZERO_FORBIDDEN", []string{"NEGATIVE_DURATION", "CLAMP_TO_ZERO_ATTEMPT"})
		}
		return refutedResult(result, "REFUTED_CLOCK_ORDER", []string{"NEGATIVE_DURATION"})
	}
	result.Decision = StateClosed
	result.Resolution = "EXACT"
	result.Reason = "DURATION_DERIVED"
	result.DurationMS = &duration
	result.AggregatePerformed = false
	return result
}

func parseEndpoint(observation Observation, start bool) (time.Time, error) {
	value := observation.CompletedAt
	field := "completed_at"
	if start {
		value = observation.StartedAt
		field = "started_at"
	}
	if value == nil || strings.TrimSpace(*value) == "" {
		if start {
			return time.Time{}, endpointError{kind: "missing-start", message: "MISSING_START_TIME"}
		}
		return time.Time{}, endpointError{kind: "missing-end", message: "MISSING_END_TIME"}
	}
	if !timezoneSuffix.MatchString(*value) {
		return time.Time{}, endpointError{kind: "malformed-timezone", message: "MALFORMED_TIMEZONE", field: field}
	}
	parsed, err := time.Parse(time.RFC3339Nano, *value)
	if err != nil {
		return time.Time{}, endpointError{kind: "malformed-timezone", message: "MALFORMED_TIMEZONE", field: field}
	}
	return parsed.UTC(), nil
}

type endpointError struct {
	kind    string
	message string
	field   string
}

func (e endpointError) Error() string {
	return e.message
}

func classifyEndpointError(result CaseResult, err error) CaseResult {
	var endpoint endpointError
	if !errors.As(err, &endpoint) {
		return refutedResult(result, "OPERATION_TIMESTAMP_MALFORMED", []string{"OPERATION_TIMESTAMP_MALFORMED"})
	}
	switch endpoint.kind {
	case "missing-start":
		return unknownResult(result, UnknownTuple{
			Stage:         "DURATION",
			Step:          "READ_START_TIME",
			Reason:        "MISSING_START_TIME",
			UnknownClass:  "DIRECT_MISSING",
			NextOperation: "RESTORE_START_TIME",
			BlockedBy:     []string{"start-time"},
		})
	case "missing-end":
		return unknownResult(result, UnknownTuple{
			Stage:         "DURATION",
			Step:          "READ_END_TIME",
			Reason:        "MISSING_END_TIME",
			UnknownClass:  "DIRECT_MISSING",
			NextOperation: "RESTORE_END_TIME",
			BlockedBy:     []string{"end-time"},
		})
	default:
		return refutedResult(result, "MALFORMED_TIMEZONE", []string{"MALFORMED_TIMEZONE"})
	}
}

func findObservation(observations []Observation, id string) (Observation, bool) {
	if id == "" {
		return Observation{}, false
	}
	for _, observation := range observations {
		if observation.ObservationID == id {
			return observation, true
		}
	}
	return Observation{}, false
}

func unknownResult(result CaseResult, tuple UnknownTuple) CaseResult {
	result.Decision = StateUnknown
	result.Resolution = "UNKNOWN"
	result.Reason = tuple.Reason
	result.Unknown = &tuple
	return result
}

func refutedResult(result CaseResult, reason string, violations []string) CaseResult {
	result.Decision = StateRefuted
	result.Resolution = "EXACT"
	result.Reason = reason
	result.Violations = violations
	return result
}

func ValidateExpected(results []CaseResult) error {
	if len(results) != 12 {
		return fmt.Errorf("result count %d, want 12", len(results))
	}
	for _, result := range results {
		if result.Decision != result.ExpectedDecision || result.Reason != result.ExpectedReason {
			return fmt.Errorf("case %s: got %s/%s, want %s/%s", result.CaseID, result.Decision, result.Reason, result.ExpectedDecision, result.ExpectedReason)
		}
		if result.Decision == StateUnknown && result.Unknown == nil {
			return fmt.Errorf("case %s: unknown tuple missing", result.CaseID)
		}
	}
	summary := SummaryFor(results)
	if summary != (Summary{Total: 12, Closed: 3, Unknown: 4, Refuted: 5}) {
		return fmt.Errorf("summary %+v does not match CLOSED=3 UNKNOWN=4 REFUTED=5", summary)
	}
	return nil
}
