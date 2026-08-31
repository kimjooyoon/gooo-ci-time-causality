package causality

import "testing"

func TestControlledCorpusContract(t *testing.T) {
	// The CI workflow is the verification authority. This unit test keeps the
	// contract executable without making local verification part of the protocol.
	results := EvaluateCases([]DurationCase{
		{
			CaseID:             "positive",
			Category:           StateClosed,
			ExpectedDecision:   StateClosed,
			ExpectedReason:     "DURATION_DERIVED",
			StartObservationID: "one",
			EndObservationID:   "one",
			Observations: []Observation{{
				ObservationID: "one",
				OperationID:   "operation",
				RunID:         "run",
				JobID:         "job",
				Provider:      "provider",
				ClockDomain:   "domain",
				StartedAt:     stringPointer("2026-08-31T00:00:00Z"),
				CompletedAt:   stringPointer("2026-08-31T00:00:01Z"),
				ArtifactID:    "artifact",
			}},
		},
	})
	if len(results) != 1 || results[0].Decision != StateClosed || results[0].DurationMS == nil || *results[0].DurationMS != 1000 {
		t.Fatalf("unexpected result: %+v", results)
	}
}

func stringPointer(value string) *string { return &value }
