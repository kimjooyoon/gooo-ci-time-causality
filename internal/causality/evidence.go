package causality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func CanonicalJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func ParseCorpus(data []byte) ([]DurationCase, error) {
	var cases []DurationCase
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for number, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var item DurationCase
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("corpus line %d: %w", number+1, err)
		}
		if item.CaseID == "" || item.ExpectedDecision == "" || len(item.Observations) == 0 {
			return nil, fmt.Errorf("corpus line %d: incomplete case", number+1)
		}
		cases = append(cases, item)
	}
	if len(cases) != 12 {
		return nil, fmt.Errorf("corpus has %d cases, want 12", len(cases))
	}
	return cases, nil
}

func BuildOperations(cases []DurationCase, results []CaseResult, fixtureData []byte) ([]byte, error) {
	decisionByCase := make(map[string]string, len(results))
	for _, result := range results {
		decisionByCase[result.CaseID] = result.Decision
	}
	var b strings.Builder
	for _, item := range cases {
		for _, observation := range item.Observations {
			line := OperationLine{
				RecordType:     "observation",
				CaseID:         item.CaseID,
				ObservationID:  observation.ObservationID,
				OperationID:    observation.OperationID,
				RunID:          observation.RunID,
				JobID:          observation.JobID,
				Provider:       observation.Provider,
				Scope:          observation.Scope,
				ClockDomain:    observation.ClockDomain,
				StartedAt:      observation.StartedAt,
				CompletedAt:    observation.CompletedAt,
				ArtifactID:     observation.ArtifactID,
				ArtifactDigest: observation.ArtifactDigest,
				Attempt:        observation.Attempt,
				Decision:       decisionByCase[item.CaseID],
			}
			data, err := json.Marshal(line)
			if err != nil {
				return nil, err
			}
			b.Write(data)
			b.WriteByte('\n')
		}
	}
	var fixture struct {
		CI struct {
			RunID    int `json:"run_id"`
			Attempts []struct {
				RunAttempt        int    `json:"run_attempt"`
				JobID             int    `json:"job_id"`
				JobStartedAt      string `json:"job_started_at"`
				JobCompletedAt    string `json:"job_completed_at"`
				ArtifactID        int    `json:"artifact_id"`
				ArtifactDigest    string `json:"artifact_digest"`
				ArtifactCreatedAt string `json:"artifact_created_at"`
				ArtifactUpdatedAt string `json:"artifact_updated_at"`
			} `json:"attempts"`
		} `json:"ci_effort_reproduction"`
	}
	if err := json.Unmarshal(fixtureData, &fixture); err != nil {
		return nil, fmt.Errorf("decode immutable retry fixture: %w", err)
	}
	if fixture.CI.RunID != 33365730015 || len(fixture.CI.Attempts) != 2 || fixture.CI.Attempts[0].RunAttempt != 1 || fixture.CI.Attempts[1].RunAttempt != 2 {
		return nil, errors.New("immutable retry fixture is not the expected attempt-1/attempt-2 lineage")
	}
	for _, attempt := range fixture.CI.Attempts {
		start := attempt.JobStartedAt
		end := attempt.JobCompletedAt
		line := OperationLine{
			RecordType:        "retry_lineage",
			LineageID:         fmt.Sprintf("ci-effort:run:%d", fixture.CI.RunID),
			CaseID:            "immutable-counterexample",
			ObservationID:     fmt.Sprintf("ci-effort-attempt-%d", attempt.RunAttempt),
			OperationID:       fmt.Sprintf("github-actions:ci-effort:run:%d", fixture.CI.RunID),
			RunID:             fmt.Sprintf("%d", fixture.CI.RunID),
			JobID:             fmt.Sprintf("%d", attempt.JobID),
			Provider:          "github-actions",
			Scope:             "ci-effort-retry",
			ClockDomain:       "github.actions.job.api.v1",
			StartedAt:         &start,
			CompletedAt:       &end,
			ArtifactID:        fmt.Sprintf("%d", attempt.ArtifactID),
			ArtifactDigest:    attempt.ArtifactDigest,
			ArtifactCreatedAt: attempt.ArtifactCreatedAt,
			ArtifactUpdatedAt: attempt.ArtifactUpdatedAt,
			Attempt:           attempt.RunAttempt,
			Decision:          StateRefuted,
		}
		data, err := json.Marshal(line)
		if err != nil {
			return nil, err
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

func CountInventory(root string) (Inventory, error) {
	result := Inventory{InputDirs: 1}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.IsDir() {
			if relative == ".git" || relative == ".github" || strings.HasPrefix(relative, ".git"+string(filepath.Separator)) || strings.HasPrefix(relative, ".github"+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			result.InputDirs++
			return nil
		}
		if relative == "README.md" {
			return nil
		}
		result.InputFiles++
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			result.PhysicalFiles++
			result.PhysicalBytes += info.Size()
		}
		extension := strings.ToLower(filepath.Ext(relative))
		if extension == ".go" {
			result.GoFiles++
			lines, err := lineCount(path)
			if err != nil {
				return err
			}
			result.GoLines += lines
		}
		if extension == ".gooo" {
			result.GoooFiles++
			lines, err := lineCount(path)
			if err != nil {
				return err
			}
			result.GoooLines += lines
		}
		return nil
	})
	return result, err
}

func lineCount(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	count := int64(1)
	for _, byteValue := range data {
		if byteValue == '\n' {
			count++
		}
	}
	if data[len(data)-1] == '\n' {
		count--
	}
	return count, nil
}

func PeakRSSKiB() int64 {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0
	}
	value := usage.Maxrss
	if runtime.GOOS == "darwin" {
		value /= 1024
	}
	return value
}

func MeasureRuntimeStats(start time.Time) RuntimeStats {
	return RuntimeStats{
		WallMS:     time.Since(start).Nanoseconds() / int64(time.Millisecond),
		PeakRSSKiB: PeakRSSKiB(),
		MeasuredBy: "go-time-and-getrusage-rusage-self",
		Scope:      "parse-ir-generate-evaluate-replay-and-inventory-before-json-serialization",
	}
}

func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func MakeOutputDigest(name string, data []byte) OutputDigest {
	return OutputDigest{Name: name, SHA256: Digest(data), Bytes: len(data)}
}
