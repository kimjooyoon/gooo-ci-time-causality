package causality

const (
	Schema       = "gooo/ci-time-causality/v1"
	ContractID   = "ci-time-causality-v1"
	StateClosed  = "CLOSED"
	StateUnknown = "UNKNOWN"
	StateRefuted = "REFUTED"
)

var ArtifactFiles = []string{
	"time-manifest.json",
	"operations.ndjson",
	"clock-domains.json",
	"duration-receipt.json",
	"replay-receipt.json",
	"time-report.md",
}

type Cell struct {
	Ordinal   int    `json:"ordinal"`
	ID        string `json:"id"`
	Activity  string `json:"activity"`
	Artifact  string `json:"artifact"`
	Evaluator string `json:"evaluator"`
}

type Contract struct {
	Schema        string   `json:"schema"`
	ContractID    string   `json:"contract_id"`
	Precedence    []string `json:"precedence"`
	Cells         []Cell   `json:"cells"`
	ArtifactFiles []string `json:"artifact_files"`
}

type Activity struct {
	Name       string
	InputType  string
	OutputType string
	SourceLine int
}

type SemanticIR struct {
	Schema       string       `json:"schema"`
	SourcePath   string       `json:"source_path"`
	SourceDigest string       `json:"source_digest"`
	Nodes        []ActivityIR `json:"nodes"`
}

type ActivityIR struct {
	Ordinal    int    `json:"ordinal"`
	CellID     string `json:"cell_id"`
	Activity   string `json:"activity"`
	SourceLine int    `json:"source_line"`
	Artifact   string `json:"artifact"`
	Evaluator  string `json:"evaluator"`
}

type Observation struct {
	ObservationID  string  `json:"observation_id"`
	OperationID    string  `json:"operation_id"`
	RunID          string  `json:"run_id"`
	JobID          string  `json:"job_id"`
	Provider       string  `json:"provider"`
	Scope          string  `json:"scope"`
	ClockDomain    string  `json:"clock_domain"`
	StartedAt      *string `json:"started_at"`
	CompletedAt    *string `json:"completed_at"`
	ArtifactID     string  `json:"artifact_id"`
	ArtifactDigest string  `json:"artifact_digest"`
	Attempt        int     `json:"attempt"`
}

type DurationCase struct {
	CaseID                  string        `json:"case_id"`
	Category                string        `json:"category"`
	Description             string        `json:"description"`
	ExpectedDecision        string        `json:"expected_decision"`
	ExpectedReason          string        `json:"expected_reason"`
	StartObservationID      string        `json:"start_observation_id"`
	EndObservationID        string        `json:"end_observation_id"`
	IndependentObservations bool          `json:"independent_observations"`
	ClampToZero             bool          `json:"clamp_to_zero"`
	Observations            []Observation `json:"observations"`
}

type UnknownTuple struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type DerivedObservation struct {
	ObservationID string `json:"observation_id"`
	OperationID   string `json:"operation_id"`
	Scope         string `json:"scope"`
	DurationMS    int64  `json:"duration_ms"`
}

type CaseResult struct {
	CaseID              string               `json:"case_id"`
	Category            string               `json:"category"`
	ExpectedDecision    string               `json:"expected_decision"`
	ExpectedReason      string               `json:"expected_reason"`
	Decision            string               `json:"decision"`
	Reason              string               `json:"reason"`
	Resolution          string               `json:"resolution"`
	Description         string               `json:"description"`
	DurationMS          *int64               `json:"duration_ms"`
	AggregatePerformed  bool                 `json:"aggregate_performed"`
	AggregateForbidden  bool                 `json:"aggregate_forbidden"`
	DerivedObservations []DerivedObservation `json:"derived_observations"`
	Violations          []string             `json:"violations"`
	Unknown             *UnknownTuple        `json:"unknown,omitempty"`
}

type Summary struct {
	Total   int `json:"total"`
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}

type RuntimeStats struct {
	WallMS     int64  `json:"wall_ms"`
	PeakRSSKiB int64  `json:"peak_rss_kib"`
	MeasuredBy string `json:"measured_by"`
	Scope      string `json:"scope"`
}

type Inventory struct {
	InputDirs     int64 `json:"input_dirs"`
	InputFiles    int64 `json:"input_files"`
	PhysicalFiles int64 `json:"physical_files"`
	PhysicalBytes int64 `json:"physical_bytes"`
	GoFiles       int64 `json:"go_files"`
	GoLines       int64 `json:"go_lines"`
	GoooFiles     int64 `json:"gooo_files"`
	GoooLines     int64 `json:"gooo_lines"`
}

type OperationLine struct {
	CaseID         string  `json:"case_id"`
	ObservationID  string  `json:"observation_id"`
	OperationID    string  `json:"operation_id"`
	RunID          string  `json:"run_id"`
	JobID          string  `json:"job_id"`
	Provider       string  `json:"provider"`
	Scope          string  `json:"scope"`
	ClockDomain    string  `json:"clock_domain"`
	StartedAt      *string `json:"started_at"`
	CompletedAt    *string `json:"completed_at"`
	ArtifactID     string  `json:"artifact_id"`
	ArtifactDigest string  `json:"artifact_digest"`
	Attempt        int     `json:"attempt"`
	Decision       string  `json:"decision"`
}

type DurationReceipt struct {
	Schema                    string       `json:"schema"`
	ContractID                string       `json:"contract_id"`
	SourceDigest              string       `json:"source_digest"`
	ImmutableFixtureDigest    string       `json:"immutable_fixture_digest"`
	IRDigest                  string       `json:"ir_digest"`
	GeneratedEvaluatorDigest  string       `json:"generated_evaluator_digest"`
	Summary                   Summary      `json:"summary"`
	Results                   []CaseResult `json:"results"`
	Runtime                   RuntimeStats `json:"runtime"`
	Inventory                 Inventory    `json:"input_inventory"`
	RepositoryWrites          int          `json:"repository_writes"`
	LocalTestExecutions       int          `json:"local_test_executions"`
	CrossProjectRequiredGates int          `json:"cross_project_required_gates"`
	AggregationRule           string       `json:"aggregation_rule"`
}

type ReplayReceipt struct {
	Schema                 string `json:"schema"`
	ContractID             string `json:"contract_id"`
	CorpusDigest           string `json:"corpus_digest"`
	FirstEvaluationDigest  string `json:"first_evaluation_digest"`
	SecondEvaluationDigest string `json:"second_evaluation_digest"`
	Deterministic          bool   `json:"deterministic"`
	ReplayCount            int    `json:"replay_count"`
	Decision               string `json:"decision"`
	Reason                 string `json:"reason"`
}

type OutputDigest struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

type Manifest struct {
	Schema                    string         `json:"schema"`
	ContractID                string         `json:"contract_id"`
	SourcePath                string         `json:"source_path"`
	SourceDigest              string         `json:"source_digest"`
	ImmutableFixtureDigest    string         `json:"immutable_fixture_digest"`
	IRDigest                  string         `json:"ir_digest"`
	GeneratedEvaluatorDigest  string         `json:"generated_evaluator_digest"`
	ActivityCount             int            `json:"activity_count"`
	CellCount                 int            `json:"cell_count"`
	ActivityCellOneToOne      bool           `json:"activity_cell_one_to_one"`
	Activities                []ActivityIR   `json:"activities"`
	ArtifactFiles             []string       `json:"artifact_files"`
	ArtifactCount             int            `json:"artifact_count"`
	Summary                   Summary        `json:"summary"`
	InputInventory            Inventory      `json:"input_inventory"`
	Runtime                   RuntimeStats   `json:"runtime"`
	OutputDigests             []OutputDigest `json:"output_digests"`
	RepositoryWrites          int            `json:"repository_writes"`
	LocalTestExecutions       int            `json:"local_test_executions"`
	CrossProjectRequiredGates int            `json:"cross_project_required_gates"`
	RootReadmeInInventory     bool           `json:"root_readme_in_inventory"`
	VerificationAuthority     string         `json:"verification_authority"`
}
