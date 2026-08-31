# gooo-ci-time-causality

`gooo-ci-time-causality` is a read-only, fail-closed observation protocol for
CI operation duration. It was created from the public counterexample in
`kimjooyoon/meta-ontology-go` PR #615, without changing that repository.

The protocol derives a duration only when the start and end timestamps belong
to the same operation identity and the same declared clock domain. A negative
duration is `REFUTED_CLOCK_ORDER`; it is never clamped to zero. Missing times,
an unknown clock domain, or missing artifact evidence are `UNKNOWN` with the
six required coordinates: `stage`, `step`, `reason`, `unknown_class`,
`next_operation`, and `blocked_by`.

The released source contains exactly twelve `.gooo` activities, one for each
denominator cell. The fixed executable corpus contains twelve cases:
`CLOSED=3`, `UNKNOWN=4`, and `REFUTED=5`. The source CI and OpenTofu
observations remain separate; no cross-run or cross-provider aggregation is
performed.

## Verification authority

GitHub Actions is the only verification authority. The workflow uses Go 1.27,
generates the semantic IR and evaluator in a caller-owned temporary directory,
replays the corpus, and uploads exactly six single-file artifacts:

* `time-manifest.json`
* `operations.ndjson`
* `clock-domains.json`
* `duration-receipt.json`
* `replay-receipt.json`
* `time-report.md`

The recorder does not write to the checkout. `repository_writes`,
`local_test_executions`, and `cross_project_required_gates` are explicit zero
observations. The root `README.md` is excluded from the input inventory by
contract; no score, average, percentage, or generalized speed claim is
reported.

See [contracts/denominator-v1.json](contracts/denominator-v1.json) for the
fixed cells, [fixtures/cases.ndjson](fixtures/cases.ndjson) for the corpus,
and [fixtures/immutable-counterexample.json](fixtures/immutable-counterexample.json)
for the public API evidence that motivated this repository.
