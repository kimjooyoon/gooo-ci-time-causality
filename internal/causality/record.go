package causality

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RunOptions struct {
	SourcePath    string
	ContractPath  string
	CorpusPath    string
	FixturePath   string
	OutputDir     string
	InventoryRoot string
}

func Run(options RunOptions) error {
	if options.SourcePath == "" || options.ContractPath == "" || options.CorpusPath == "" || options.FixturePath == "" || options.OutputDir == "" {
		return errors.New("source, contract, corpus, fixture, and output are required")
	}
	if options.InventoryRoot == "" {
		options.InventoryRoot = "."
	}
	if err := ensureEmptyOutput(options.OutputDir); err != nil {
		return err
	}
	started := time.Now()
	sourceData, err := os.ReadFile(options.SourcePath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	contractData, err := os.ReadFile(options.ContractPath)
	if err != nil {
		return fmt.Errorf("read contract: %w", err)
	}
	corpusData, err := os.ReadFile(options.CorpusPath)
	if err != nil {
		return fmt.Errorf("read corpus: %w", err)
	}
	fixtureData, err := os.ReadFile(options.FixturePath)
	if err != nil {
		return fmt.Errorf("read immutable fixture: %w", err)
	}
	contract, err := LoadContract(contractData)
	if err != nil {
		return err
	}
	activities, err := ParseGooo(sourceData)
	if err != nil {
		return err
	}
	sourceDigest := Digest(sourceData)
	ir, err := BuildIR(options.SourcePath, sourceDigest, activities, contract)
	if err != nil {
		return err
	}
	irData, err := json.MarshalIndent(ir, "", "  ")
	if err != nil {
		return err
	}
	irDigest := Digest(irData)
	evaluatorData, err := GenerateEvaluator(ir)
	if err != nil {
		return err
	}
	evaluatorDigest := Digest(evaluatorData)
	workingDir := filepath.Join(filepath.Dir(options.OutputDir), "working")
	if err := WriteFile(filepath.Join(workingDir, "ir.json"), irData); err != nil {
		return fmt.Errorf("write semantic IR: %w", err)
	}
	if err := WriteFile(filepath.Join(workingDir, "generated", "evaluator.go"), evaluatorData); err != nil {
		return fmt.Errorf("write generated evaluator: %w", err)
	}
	cases, err := ParseCorpus(corpusData)
	if err != nil {
		return err
	}
	firstResults := EvaluateCases(cases)
	if err := ValidateExpected(firstResults); err != nil {
		return err
	}
	secondResults := EvaluateCases(cases)
	if err := ValidateExpected(secondResults); err != nil {
		return err
	}
	firstCanonical, err := CanonicalJSON(firstResults)
	if err != nil {
		return err
	}
	secondCanonical, err := CanonicalJSON(secondResults)
	if err != nil {
		return err
	}
	replay := ReplayReceipt{
		Schema:                 "gooo/ci-time-causality/replay/v1",
		ContractID:             ContractID,
		CorpusDigest:           Digest(corpusData),
		FirstEvaluationDigest:  Digest(firstCanonical),
		SecondEvaluationDigest: Digest(secondCanonical),
		Deterministic:          string(firstCanonical) == string(secondCanonical),
		ReplayCount:            2,
		Decision:               StateClosed,
		Reason:                 "DETERMINISTIC_CASE_REPLAY",
	}
	if !replay.Deterministic {
		return errors.New("deterministic replay mismatch")
	}
	replayData, err := json.MarshalIndent(replay, "", "  ")
	if err != nil {
		return err
	}
	operationsData, err := BuildOperations(cases, firstResults)
	if err != nil {
		return err
	}
	clockDomainsData, err := clockDomainsJSON()
	if err != nil {
		return err
	}
	inventory, err := CountInventory(options.InventoryRoot)
	if err != nil {
		return fmt.Errorf("inventory: %w", err)
	}
	runtimeStats := MeasureRuntimeStats(started)
	summary := SummaryFor(firstResults)
	duration := DurationReceipt{
		Schema:                    "gooo/ci-time-causality/duration-receipt/v1",
		ContractID:                ContractID,
		SourceDigest:              sourceDigest,
		ImmutableFixtureDigest:    Digest(fixtureData),
		IRDigest:                  irDigest,
		GeneratedEvaluatorDigest:  evaluatorDigest,
		Summary:                   summary,
		Results:                   firstResults,
		Runtime:                   runtimeStats,
		Inventory:                 inventory,
		RepositoryWrites:          0,
		LocalTestExecutions:       0,
		CrossProjectRequiredGates: 0,
		AggregationRule:           "Only same operation_id, run_id, job_id, provider, and clock_domain may form one duration; source-ci and opentofu remain separate observations.",
	}
	durationData, err := json.MarshalIndent(duration, "", "  ")
	if err != nil {
		return err
	}
	outputDigests := []OutputDigest{
		MakeOutputDigest("operations.ndjson", operationsData),
		MakeOutputDigest("clock-domains.json", clockDomainsData),
		MakeOutputDigest("duration-receipt.json", durationData),
		MakeOutputDigest("replay-receipt.json", replayData),
	}
	manifest := Manifest{
		Schema:                    "gooo/ci-time-causality/time-manifest/v1",
		ContractID:                ContractID,
		SourcePath:                options.SourcePath,
		SourceDigest:              sourceDigest,
		ImmutableFixtureDigest:    Digest(fixtureData),
		IRDigest:                  irDigest,
		GeneratedEvaluatorDigest:  evaluatorDigest,
		ActivityCount:             len(ir.Nodes),
		CellCount:                 len(contract.Cells),
		ActivityCellOneToOne:      true,
		Activities:                ir.Nodes,
		ArtifactFiles:             append([]string(nil), ArtifactFiles...),
		ArtifactCount:             len(ArtifactFiles),
		Summary:                   summary,
		InputInventory:            inventory,
		Runtime:                   runtimeStats,
		OutputDigests:             outputDigests,
		RepositoryWrites:          0,
		LocalTestExecutions:       0,
		CrossProjectRequiredGates: 0,
		RootReadmeInInventory:     false,
		VerificationAuthority:     "github-actions",
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifestDigest := Digest(manifestData)
	reportData := []byte(buildHumanReport(manifest, manifestDigest, outputDigests, replay, firstResults))
	files := map[string][]byte{
		"time-manifest.json":    manifestData,
		"operations.ndjson":     operationsData,
		"clock-domains.json":    clockDomainsData,
		"duration-receipt.json": durationData,
		"replay-receipt.json":   replayData,
		"time-report.md":        reportData,
	}
	for _, name := range ArtifactFiles {
		if err := WriteFile(filepath.Join(options.OutputDir, name), files[name]); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

func ensureEmptyOutput(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory is not empty: %s", path)
	}
	return nil
}

type clockDomain struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	ResolutionMS int64  `json:"resolution_ms"`
	Comparable   string `json:"comparable"`
}

type clockDomainDocument struct {
	Schema  string        `json:"schema"`
	Rule    string        `json:"rule"`
	Domains []clockDomain `json:"domains"`
}

func clockDomainsJSON() ([]byte, error) {
	document := clockDomainDocument{
		Schema: "gooo/ci-time-causality/clock-domains/v1",
		Rule:   "Duration subtraction requires equal clock_domain and equal operation identity; wall-clock values are normalized to UTC only after timezone validation.",
		Domains: []clockDomain{
			{ID: "fixture.utc.v1", Source: "synthetic corpus", ResolutionMS: 1, Comparable: "only within the same operation"},
			{ID: "github.actions.run.api.v1", Source: "GitHub REST run timestamps", ResolutionMS: 1000, Comparable: "only within the same run operation"},
		},
	}
	return json.MarshalIndent(document, "", "  ")
}

func buildHumanReport(manifest Manifest, manifestDigest string, digests []OutputDigest, replay ReplayReceipt, results []CaseResult) string {
	summary := manifest.Summary
	var b strings.Builder
	b.WriteString("# CI time causality report\n\n")
	b.WriteString("decision: REFUTED (controlled corpus contains five expected refutations)\n")
	fmt.Fprintf(&b, "contract: %s; denominator_cells: %d; released_gooo_activities: %d; one_to_one: %t\n", manifest.ContractID, manifest.CellCount, manifest.ActivityCount, manifest.ActivityCellOneToOne)
	fmt.Fprintf(&b, "cases: total=%d CLOSED=%d UNKNOWN=%d REFUTED=%d; precedence=REFUTED>UNKNOWN>CLOSED\n", summary.Total, summary.Closed, summary.Unknown, summary.Refuted)
	fmt.Fprintf(&b, "runtime: wall_ms=%d peak_rss_kib=%d measured_by=%s\n", manifest.Runtime.WallMS, manifest.Runtime.PeakRSSKiB, manifest.Runtime.MeasuredBy)
	fmt.Fprintf(&b, "input_inventory: dirs=%d files=%d physical_files=%d physical_bytes=%d Go_files=%d Go_lines=%d Gooo_files=%d Gooo_lines=%d root_README_included=false\n", manifest.InputInventory.InputDirs, manifest.InputInventory.InputFiles, manifest.InputInventory.PhysicalFiles, manifest.InputInventory.PhysicalBytes, manifest.InputInventory.GoFiles, manifest.InputInventory.GoLines, manifest.InputInventory.GoooFiles, manifest.InputInventory.GoooLines)
	b.WriteString("authority: verification=github-actions Go=1.27 repository_writes=0 local_test_executions=0 cross_project_required_gates=0\n")
	b.WriteString("duration rule: start and end are derived only from the same operation identity and clock domain; negative duration is REFUTED_CLOCK_ORDER; clamp-to-zero is forbidden.\n")
	b.WriteString("aggregation: source CI and OpenTofu are separate observations; no cross-run, cross-job, or cross-provider subtraction is performed.\n\n")
	b.WriteString("## Case results\n\n")
	for _, result := range results {
		b.WriteString("- ")
		b.WriteString(result.CaseID)
		b.WriteString(": ")
		b.WriteString(result.Decision)
		b.WriteString(" / ")
		b.WriteString(result.Reason)
		if result.DurationMS != nil {
			fmt.Fprintf(&b, " / duration_ms=%d", *result.DurationMS)
		}
		if result.Unknown != nil {
			fmt.Fprintf(&b, " / unknown(stage=%s,step=%s,reason=%s,unknown_class=%s,next_operation=%s,blocked_by=%s)", result.Unknown.Stage, result.Unknown.Step, result.Unknown.Reason, result.Unknown.UnknownClass, strings.Join(result.Unknown.BlockedBy, ","), result.Unknown.NextOperation)
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n## Replay and output digests\n\n")
	fmt.Fprintf(&b, "replay: deterministic=%t replay_count=%d first=%s second=%s\n", replay.Deterministic, replay.ReplayCount, replay.FirstEvaluationDigest, replay.SecondEvaluationDigest)
	fmt.Fprintf(&b, "time-manifest.json: %s\n", manifestDigest)
	for _, digest := range digests {
		fmt.Fprintf(&b, "%s: %s bytes=%d\n", digest.Name, digest.SHA256, digest.Bytes)
	}
	b.WriteString("time-report.md: digest is intentionally resolved by the GitHub Actions artifact API after upload (self-digest is not embedded).\n")
	b.WriteString("\nNo score, average, percentage, or generalized speed claim is emitted.\n")
	return b.String()
}
