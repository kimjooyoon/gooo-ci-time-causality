# CI time causality protocol v1

## Identity boundary

An operation identity is the tuple `(operation_id, run_id, job_id,
provider)`. A duration can be derived only from the `started_at` and
`completed_at` values of that identity in one declared `clock_domain`.
Timezone syntax is validated before conversion to UTC. A different operation,
run, job, provider, or clock domain is never silently combined.

The source CI run and the OpenTofu run are separate observations. Each may
produce its own exact duration. They are not summed, averaged, compared as a
speed claim, or used as endpoints of one interval.

## Lattice

The case state precedence is `REFUTED > UNKNOWN > CLOSED`.

* `CLOSED` means the exact non-negative duration was derived, or independent
  source observations were each derived.
* `UNKNOWN` means required evidence is absent or the clock domain is not
  declared. It carries all six coordinates: `stage`, `step`, `reason`,
  `unknown_class`, `next_operation`, and `blocked_by`.
* `REFUTED` is fail-closed evidence. Negative order is
  `REFUTED_CLOCK_ORDER`; malformed timezone and boundary subtraction are also
  refuted. A negative duration is never changed to zero.

The corpus deliberately contains all twelve executable cases and is expected
to resolve as three closed, four unknown, and five refuted cases. Those are
integer observations of this corpus, not a score or percentage.

## Provenance and replay

The public API snapshot in `fixtures/immutable-counterexample.json` records the
exact PR head, CI run, OpenTofu run, CI-effort retry jobs, artifact IDs,
artifact SHA-256 digests, and timestamp tuple. The old Guardian failure is
explicitly outside the fixture.

The recorder parses `.gooo` into a twelve-node IR, emits a generated evaluator
in the caller-owned temporary directory, evaluates the fixed corpus twice, and
compares canonical result digests. The checkout is not written by the
recorder. Six single-file GitHub Actions artifacts are the only published
record outputs.
