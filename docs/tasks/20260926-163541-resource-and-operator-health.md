# Resource Bounds And Operator Workflow Health

Created: 2026-09-26 16:35:41 (local timestamp)

## Objective And Product Context

s3proxy provides a virtual S3 namespace over configured S3-compatible backends. Its business workflows are ordinary object access, tenant/bucket namespace isolation, replicated writes, and predictable single-backend listings. Reliability under fragmented I/O and understandable routing configuration directly support those workflows.

This follow-up implements two evidenced resource defects and one operator feature. It addresses availability/security hardening, measured performance, usability, and encapsulation/testability through a narrow safe reporting boundary. It is not a general rewrite or a claim of exhaustive security coverage.

## Analysis Coverage, Baseline, And Related Work

- Inspected product/design documentation, CLI, app composition, config compilation/types, request normalization, virtual bucket service, replay storage, listing XML transformation, and relevant callers/tests. Sequential discovery agents additionally reviewed auth/forwarding boundaries and operator documentation.
- Existing working tree contains substantial user/prior-agent changes, including untracked tests and namespace code. Preserve all of it; no resets, commits, broad formatting, or cleanup.
- Deduplicated against `20260804-133206-codebase-health-remediation.md`, `20260823-113640-codebase-health-follow-up.md`, `20260926-145643-s3-workflow-health.md`, `20260926-151017-sandbox-compose-isolation.md`, and `20260926-155056-final-review-follow-ups.md` in this directory. These tasks extend completed replay/listing hardening, not repeat it.
- Baseline `make vet test` passed on 2026-09-26 before this plan was created (unit packages cached). No baseline performance claim: agents must collect before/after measurements using their new fixtures.
- Not covered comprehensively: dependency audit, cloud-provider conformance, deployment load testing, release artifacts, and live sandbox behavior. No live stack is needed for these local algorithm/CLI changes; in-process workflow tests and full unit/race/build checks are required.
- Multipart uploads, DB-backed credential lifecycle, merged listings, and hot reload remain intentional V1 non-goals, as recorded in `docs/design.md`. They require a separate product/state-model decision; partial implementations here would expand the protocol without satisfying their business contracts.
- A process-wide listing transformation budget remains a documented limitation (`docs/design.md`, Listing Semantics). This plan fixes confirmed quadratic work within existing response bounds; concurrent response memory budgeting needs separate capacity measurements.

## Execution Rules And Shared Verification

- Run tasks in order below, each in a **fresh, separate sub-agent session**, sequentially. No nested agents. Coordinator owns sequencing and final closeout.
- Set `Status: in_progress` when starting; set `completed` only after acceptance and verification pass, with `Completion evidence` (changed paths, commands/results, measurements where required). Record blockers explicitly rather than claiming completion.
- P1 = concrete availability/resource-bound failure on supported uploads. P2 = bounded resilience or operator improvement. Final review inherits the release-blocking importance of preceding tasks.
- All commands run from `<repo-root>`. Go/toolchain dependencies are already installed. After each nontrivial implementation run `make vet test`; final reviewer runs `make test-race` and `make build` as well.
- Performance tests compare allocations/scaling, without brittle timing assertions. Keep regression tests with their fixes.
- Update relevant design/operator docs for new behavior and allowances. **Never modify `CHANGELOG.md`**; this task record and relevant docs hold implementation/release-facing notes. Starting SHA-256: `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`.
- Follow `AGENTS.md`; preserve streaming object reads, 413/503 replay failure semantics, listing limits/namespace checks, diagnostic confidentiality, and caller-owned replay lifetimes.

### Task REPLAY-02: Bound Replay Allocation Under Short Reads

Status: completed

Kind: defect

Priority: P1 — supported fragmented uploads defeat the practical retained-memory bound.

Suggested agent: Go I/O and resource-accounting implementer.

Dependencies: none.

Primary ownership: `internal/replaybody/replaybody.go`, its tests/benchmarks, replay-budget documentation.

Finding: `Budget.readUnknown` allocates a chunk and appends a slice descriptor for every positive source `Read`, charging only the payload bytes. A one-byte reader produces one descriptor per byte (24 bytes per slice descriptor on 64-bit Go, before backing allocations/growth). Existing benchmarks use filling `bytes.Reader` reads. Completed RESOURCE-01 in the August follow-up addressed other allocation/race issues, but does not cover this amplification.

References:

- `internal/replaybody/replaybody.go:158-195` (`Budget.readUnknown`).
- `internal/replaybody/replaybody_benchmark_test.go` (`BenchmarkEnsureReplay`).
- `internal/backend/s3/client.go` (`prepareSourceBody`); `internal/dispatch/dispatch.go` (replay before fan-out).

Requirements:

1. Coalesce short reads so retained chunk/metadata count scales with payload divided by chunk size, not the number of reads. Keep retained allocation accounting bounded; document any fixed scratch/final-chunk allowance precisely.
2. Preserve exact bytes, GetBody/reset/release behavior, cancellation, close ownership, and per-request/aggregate errors. Cover EOF with data and errors after partial progress; avoid a new busy loop on repeated zero-progress reads.
3. Add short-read/irregular-read regressions and a benchmark that can run on both old and corrected implementations. Record comparable before/after measurements.

Acceptance criteria:

- One-byte reads reconstruct the original multi-chunk body and support concurrent independent replays with bounded retained chunks/capacity.
- Exact size limit succeeds, overflow is 413-class, aggregate exhaustion is 503-class, and failure/cancellation/closure releases reservations.
- Allocation measurements demonstrate elimination of per-byte chunk allocation. Known-length and ordinary unknown-length behavior remains passing.

Verification: targeted replay tests; `go test -race ./internal/replaybody ./internal/dispatch ./internal/httpapi`; `go test -run '^$' -bench BenchmarkEnsureReplay -benchmem ./internal/replaybody`; shared `make vet test`.

Completion evidence:

- Changed: `internal/replaybody/replaybody.go`, `internal/replaybody/replaybody_benchmark_test.go`, new `internal/replaybody/shortread_test.go`, replay storage section in `docs/design.md`, runtime memory-sizing guidance in `website/docs/operations.md`, and this task's status/evidence only. Existing dirty/untracked work was preserved; no commits, resets, or nested agents.
- Implementation: unknown-length reads fill a reusable scratch buffer before copying full chunks or the final partial chunk into immutable, exact-capacity slices. Each positive read reserves its bytes immediately, including incomplete chunks. N payload bytes retain exactly ceil(N / 32768) chunks with total payload capacity N; metadata scales with chunks rather than read calls. EOF-with-data observes cancellation; 100 consecutive `(0, nil)` reads return wrapped `io.ErrNoProgress`, with positive progress resetting the counter.
- Exact allowance: one transient **32,768-byte scratch buffer per active unknown-length buffering operation**, outside payload accounting. **Zero extra retained final-chunk capacity**. Slice descriptors (24 bytes each on 64-bit Go, including spare descriptor capacity), request/replay bookkeeping, and allocator rounding remain additional overhead, documented explicitly; the aggregate payload budget is not an RSS limit.
- Regression-first: added the benchmark and measured the original reader before changing production code. Then `go test -count=1 -run '^TestUnknownShortRead' ./internal/replaybody` failed as expected: five chunk-bound cases (98,441 one-byte chunks instead of four for the multi-chunk fixture; 65 irregular chunks instead of four; 32,768 instead of one at a full-chunk boundary), cancellation-with-data/EOF incorrectly succeeding, and missing no-progress termination. No whole-file restoration or reset was used.
- Coverage: nonuniform multi-chunk bytes, one-byte and irregular reads including intermittent empty reads, EOF with/without final data, full-chunk and empty bodies, bounded chunk count/metadata capacity and exact retained payload capacity, eight concurrent independent replays, reset after partial consumption, acquired-reader validity after release, caller versus independent-reader closure, exact request/aggregate limits, overflow/exhaustion while another reservation stays intact, data-plus-error and error-after-progress, close errors, cancellation (including data/EOF), immediate partial charging, and empty-read counter reset. Existing known-length and blocked-source cancellation tests pass. Existing handler tests verify the preserved `413 EntityTooLarge` / `503 SlowDown` mappings.
- Verification passed: `go test -count=1 ./internal/replaybody`; `go test -race ./internal/replaybody ./internal/dispatch ./internal/httpapi`; `make vet test` (vet and all unit packages); `gofmt -l internal/replaybody/replaybody.go internal/replaybody/replaybody_benchmark_test.go internal/replaybody/shortread_test.go` (no output). Session-path `git diff --check` and no-index whitespace checks for the new regression/task files pass.
- Both measurements used exactly `go test -run '^$' -bench BenchmarkEnsureReplay -benchmem ./internal/replaybody`, with the same fixtures excluding payload setup via `ResetTimer`, on Go 1.26.6, linux/amd64, Intel Core i9-13950HX, benchmark suffix `-32`. These are single-run observations, not timing thresholds; unchanged known/ordinary-unknown allocation counts provide controls despite timing variation.

| Fixture             | Before B/op | After B/op | Before allocs/op | After allocs/op | Before ns/op | After ns/op |
| ------------------- | ----------: | ---------: | ---------------: | --------------: | -----------: | ----------: |
| Known/1KiB          |       2,080 |      2,080 |               16 |              16 |        3,615 |       4,077 |
| Known/1MiB          |   1,049,652 |  1,049,648 |               16 |              16 |    1,329,645 |   1,645,572 |
| Known/Max (32MiB)   |  33,555,493 | 33,555,517 |               16 |              16 |   12,298,307 |  26,729,880 |
| Unknown/1KiB        |      34,872 |     34,872 |               18 |              18 |       37,156 |      43,007 |
| Unknown/1MiB        |   1,084,046 |  1,084,052 |               54 |              54 |    1,332,405 |   1,865,481 |
| Unknown/Max (32MiB) |  33,647,627 | 33,647,633 |            1,051 |           1,051 |   15,491,720 |  20,415,778 |
| OneByte/1KiB        |      94,297 |     34,952 |            1,052 |              19 |      210,929 |     109,439 |
| OneByte/64KiB       |   8,154,249 |     99,512 |           65,579 |              21 |   18,707,308 |   3,981,217 |
| OneByte/1MiB        | 129,002,653 |  1,084,120 |        1,048,631 |              55 |  240,339,589 |  60,428,162 |
| Irregular/1MiB      |   1,142,943 |  1,084,128 |              711 |              55 |    1,515,145 |   1,747,240 |

- Result: 1 MiB one-byte allocation volume fell approximately 119-fold; allocation count is now chunk-scaled and matches irregular reads at 55 allocations/op. No acceptance blockers. `CHANGELOG.md` SHA-256 remains `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`. XML-02, CLI-02, and FINAL-02 remain pending.

### Task XML-02: Make Fragmented Listing Text Accumulation Linear

Status: completed

Kind: defect

Priority: P2 — disproportionate CPU/allocation work from bounded upstream listing responses.

Suggested agent: XML parsing/performance implementer.

Dependencies: REPLAY-02 (sequential execution; no semantic dependency).

Primary ownership: `internal/namespace/response.go`, focused tests/benchmarks, listing rationale in `docs/design.md`.

Finding: `parseResult` appends every character-data token with string concatenation. Comments/CDATA can split a large leaf into many tokens without increasing the element count, making repeated copies quadratic under the existing 8 MiB/50,000-element limits. The five-second deadline is not a cumulative allocation bound. This concerns upstream-response resilience, not demonstrated client XML injection. Completed LIST-01 did not test text fragmentation.

References:

- `internal/namespace/response.go:105-172` (`parseResult`, especially CharData handling).
- `internal/namespace/response_test.go` (current transformation bounds).
- `internal/backend/s3/client.go` (successful ListObjectsV2 transformation).

Requirements:

1. Accumulate leaf text using amortized-linear storage and finalize once per leaf, keeping mutable parse state local and the translated element model straightforward.
2. Preserve plain text, XML entity, comment and CDATA semantics; all namespace, duplicate-field, element/depth/input/output, cancellation, and body closure checks must remain effective.
3. Add semantically equivalent fragmented/unfragmented fixtures and scaling benchmarks with before/after allocation evidence.

Acceptance criteria:

- Equivalent text fragmented by comments or CDATA transforms to the same virtual listing output.
- Measurements show approximately linear allocation growth for increasing fragmented text sizes; no timing-threshold unit tests.
- Malformed XML and out-of-namespace content still fail closed, including fragmented content; focused existing workflow tests pass.

Verification: `go test ./internal/namespace ./internal/backend/s3 ./internal/httpapi`; `go test -race ./internal/namespace ./internal/backend/s3`; new focused benchmarks with `-benchmem`; shared `make vet test`.

Completion evidence:

- Confirmed REPLAY-02 completed before starting; this fresh implementer session changed only XML-02's status. No nested agents, commits, resets, whole-file restorations, or changes to `CHANGELOG.md`. Existing dirty/untracked files and prior documentation edits were preserved.
- Changed: `internal/namespace/response.go`, new `internal/namespace/response_fragmentation_test.go`, leaf accumulation rationale in `docs/design.md`, and this task's status/evidence.
- Implementation: one local `strings.Builder` copies character-data tokens into amortized-linear storage; each leaf's closing tag finalizes its string and resets the builder. Valid leaves cannot nest, so mutable state stays entirely inside `parseResult`, with no builder in the element model or cross-request state. Existing validation, per-token context checks, size/depth/element limits, encoding, and exactly-once body closure/deadline paths are retained.
- Regression coverage: byte-identical virtual output for plain, comment-fragmented, CDATA-fragmented, and mixed text, including one-byte source reads, named/numeric entities, Unicode, XML metacharacters, whitespace, opaque pagination tokens, multiple entries, empty leaves, repeated checksum fields, and four-level owner metadata. Parallel cases exercise independent parser state. Each mode rejects malformed/truncated XML, mismatched tags, unterminated comments/CDATA, invalid entities, foreign/reset XML namespaces, nested foreign fields, duplicate singleton fields, container/trailing text, and sibling keys/common prefixes; failure returns no partial output and closes exactly once.
- Before production changes, `go test -count=1 -run '^TestTransformFragmentedText' ./internal/namespace` passed with the finalized fixtures, and the baseline benchmark below exposed quadratic allocation. The same fixture file and benchmark command were used after the fix; no timing thresholds were added to unit tests.
- Verification passed: `go test -count=1 ./internal/namespace ./internal/backend/s3 ./internal/httpapi`; `go test -race ./internal/namespace ./internal/backend/s3`; `make vet test` (vet and all unit packages). Existing transformation-bound, namespace, cancellation/deadline, close-error, backend and HTTP workflow tests pass. `gofmt -l` on both changed Go files produced no output; session-path whitespace checks passed.
- Before/after command: `go test -run '^$' -bench '^BenchmarkParseResultFragmentedText$' -benchmem ./internal/namespace`, Go 1.26.6, linux/amd64, Intel Core i9-13950HX, suffix `-32`. Each fixture is one ETag leaf with 1/8/64 KiB decoded ASCII text, plain or split into eight-character fragments; mixed alternates plain/CDATA with comments. Fixture construction is excluded with `ResetTimer`; parsing and exact text verification are measured. Single-run observations follow, not timing guarantees.

| Fixture       | Before B/op | After B/op | Before allocs/op | After allocs/op | Before ns/op | After ns/op |
| ------------- | ----------: | ---------: | ---------------: | --------------: | -----------: | ----------: |
| Plain/1KiB    |       6,200 |      6,200 |               80 |              80 |       31,183 |      34,790 |
| Plain/8KiB    |      27,704 |     27,704 |               83 |              83 |      159,430 |     178,871 |
| Plain/64KiB   |     199,742 |    199,744 |               86 |              86 |    1,260,522 |   1,158,686 |
| Comment/1KiB  |      79,756 |     12,696 |              585 |             339 |      178,975 |     123,507 |
| Comment/8KiB  |   4,531,331 |     86,683 |            4,170 |           2,137 |    4,040,895 |     945,350 |
| Comment/64KiB | 289,274,006 |    681,925 |           32,963 |          16,480 |  186,348,057 |   7,546,127 |
| CDATA/1KiB    |      76,684 |      9,624 |              457 |             211 |      106,090 |      64,927 |
| CDATA/8KiB    |   4,506,753 |     62,105 |            3,147 |           1,113 |    2,800,860 |     497,954 |
| CDATA/64KiB   | 289,073,072 |    485,276 |           24,740 |           8,288 |  177,994,083 |   4,108,147 |
| Mixed/1KiB    |      79,753 |     12,696 |              585 |             339 |      269,171 |     207,384 |
| Mixed/8KiB    |   4,531,271 |     86,682 |            4,170 |           2,137 |    4,200,862 |   1,105,363 |
| Mixed/64KiB   | 289,268,017 |    681,902 |           32,916 |          16,480 |  192,996,950 |   8,916,012 |

- Result: 8× larger fragmented text (8→64 KiB) now allocates approximately 7.8–7.9× as many bytes, versus approximately 64× before. At 64 KiB, allocation volume falls approximately 424× for comments/mixed and 596× for CDATA. XML decoder token allocations still scale with fragment count; the plain-text allocation control is unchanged within measurement rounding. No acceptance blockers. The documented per-response rather than aggregate listing budget remains applicable. `CHANGELOG.md` SHA-256 remains `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`; CLI-02 and FINAL-02 remain pending.

### Task CLI-02: Add Safe Offline Route-Topology Inspection

Status: completed

Kind: improvement

Priority: P2 — help operators review replication/order/defaults before deployment without serving traffic.

Suggested agent: CLI/configuration UX implementer.

Dependencies: XML-02 (sequential execution; no semantic dependency).

Primary ownership: `cmd/s3proxy`, a narrow configuration-report module if useful, `README.md`, `docs/design.md`, `website/docs/operations.md`.

Finding: CLI registers only serve/validate/version; validation prints only `config is valid`. Operators cannot inspect normalized ordered topology, despite route/destination order and dispatch/read defaults controlling business behavior. `docs/design.md` explicitly lists `routes --config` as an optional future feature. Prior side-effect-free validation and runtime ownership tasks do not implement this report.

References:

- `cmd/s3proxy/main.go` (command registration and `newValidateCommand`).
- `internal/config/load.go:305-346` (route normalization/defaults).
- `internal/router/resolve.go` (ordered route/destination selection).
- `website/docs/operations.md` (rollout/troubleshooting); `website/docs/configuration.md` (public labels and sensitive diagnostic values).

Requirements:

1. Add `s3proxy routes --config PATH` / `-c PATH` producing deterministic JSON from fully validated config, without app construction, listener binding, backend calls, or runtime logs.
2. Use an explicit allowlisted report DTO: route ordinal/label, resolved parser label/kind, operation names, ordered resolved target labels, dispatch/on_match/read_preference. Specify the meaning of empty operation filters consistently with runtime behavior.
3. Emit only source-literal declaration identities and validated finite-vocabulary fields. Resolve refs to declaration labels. Exclude credentials, endpoints, regions, parser match strings, regexes, rewrites/templates, visible bucket values, environment names/values, and raw reference expressions. Never serialize Runtime/Route wholesale.
4. Invalid config yields safe existing errors, nonzero exit and no partial report. Reject missing config/positional arguments. Propagate output failures.
5. Document report schema, examples, public-label confidentiality boundary, and that this is configured topology, not simulated request routing or a health/authorization check.

Acceptance criteria:

- Repeated invocations yield identical JSON preserving route/destination order and resolved refs/defaults.
- Tests exercise multiple routes, qualified refs (including environment-derived qualified prefixes), operation filters/default semantics, argument errors, invalid config and output failures.
- Real CLI stdout/stderr tests verify secret sentinels in arbitrary string fields cannot leak on success or failure; no binding/network side effects.
- Dedicated DTO prevents future config fields from automatically entering public output; docs and implementation agree.

Verification: `go test ./cmd/s3proxy ./internal/config ./internal/app ./internal/router`; shared `make vet test`.

Completion evidence:

- Confirmed XML-02 completed before starting in this fresh isolated implementer session; CLI-02 was marked `in_progress` before implementation and `completed` only after acceptance verification. Changed no other task status. No nested agents, commits, resets, or edits to `CHANGELOG.md`; preserved the inventoried dirty/untracked user and prior-agent work.
- Changed: command registration in `cmd/s3proxy/main.go`; new `cmd/s3proxy/routes.go` and `cmd/s3proxy/routes_test.go`; CLI/schema/confidentiality/operator documentation in `README.md`, `docs/design.md`, and `website/docs/operations.md`; this task's status/evidence.
- Implementation: `routes --config PATH` / `routes -c PATH` uses `config.LoadFile` for full parsing, compilation, and validation before building a private, explicitly allowlisted DTO. It does not call app construction or runtime/network/listener APIs. Output is two-space-indented JSON with a trailing newline; the `routes` array retains declaration order and one-based ordinals, and operation/destination arrays retain configured order. Parser and target identities come from resolved declaration objects, never reference expressions or prefixes. Empty topology is `{"routes": []}` (pretty-printed).
- Contract: report fields are exactly route `ordinal`, `label`, `parser` (`label`, `kind`), `operations`, `destinations`, `dispatch`, `on_match`, and `read_preference`. Only source-literal declaration identities and validated finite-vocabulary strings are exposed. Empty/omitted/wildcard route operation filters are invalid; matching is exact and case-sensitive, with no implicit all-operations filter. Omitted or empty read preference resolves to `first`; dispatch/on_match remain required. Documentation distinguishes configured topology from request simulation, effective read selection, health, and authorization, and explains local `ListBuckets` handling and public labels/diagnostic filenames.
- Real CLI acceptance: subprocess tests build and invoke the actual binary through `main` and Cobra. A four-route/two-target fixture covers all parser kinds, all read preferences, fan-out/continue, qualified/bare refs, deliberately nonalphabetical route/operation/destination order, and all configurable operations. Three repeated runs per literal/env variant with both flags produce byte-identical JSON checked against an independent exact-schema JSON expectation (including no extra fields). Environment-derived qualified parser/target/credential/route prefixes contain private sentinels yet resolve only to public labels; valid environment enums/operations and the empty read-preference default are covered separately.
- Confidentiality/validation: successful literal and environment fixtures place sentinels in client/backend credentials, endpoint userinfo/path/query/fragment, region, host addressing, all four parser match fields, every rewrite string/template, visible buckets and policy/reference attributes. Failure cases cover URL, target/listener durations, regex, template, parser/target/credential/policy/bucket refs, auth/dispatch/match/read enums, operations, HCL evaluated duplicate keys and type conversion, missing credentials, parse errors, empty/omitted/wildcard/duplicate operations, and normalized duplicate destinations. Compiler/validator cases also exercise literal invalid values. Each invalid run exits 1, emits no report, retains useful safe stderr, and excludes private fragments/environment names from both streams.
- Offline/error acceptance: real CLI succeeds with an already-occupied listener and with a non-bindable address; a connection-counting local backend observes zero connections across success and failure cases. Exact JSON/empty success stderr excludes app/runtime logs. Missing config, missing flag value, empty path, positional args, and unreadable config exit 1 with empty stdout. Real `/dev/full` output failure exits 1 with safe write diagnostics (executed on Linux); injected writers verify wrapped error identity, partial failed writes, and `io.ErrShortWrite`. Invalid config is fully rejected before output; device write failures may leave partial bytes, as documented.
- Verification passed: `go test -count=1 -run '^TestRoutes' ./cmd/s3proxy` during initial implementation; final `go test -count=1 ./cmd/s3proxy ./internal/config ./internal/app ./internal/router`; `make vet test` (vet and all 17 unit packages). `gofmt -l cmd/s3proxy/main.go cmd/s3proxy/routes.go cmd/s3proxy/routes_test.go` produced no output. Session-path `git diff --check` and no-index whitespace checks of the new Go/task files passed.
- Actual example: `go run ./cmd/s3proxy routes -c <repo-root>/tmp/s3proxy-cli02-example.hcl` exited 0 and printed exactly the example in `website/docs/operations.md#offline-route-topology`: route `all`, ordinal 1, parser `{label: all, kind: path_prefix}`, operations `[GetObject]`, destinations `[primary]`, dispatch `first`, on_match `stop`, read_preference `first`. The fixture uses fully qualified parser/target refs and synthetic credentials; no endpoints or credentials appear in output.
- Result: all CLI-02 acceptance criteria passed; no blockers. `CHANGELOG.md` SHA-256 remains `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`. FINAL-02 remains pending for its separate independent review/full race/build gate.

### Task FINAL-02: Independently Review Acceptance And Integration

Status: completed

Review start: fresh independent reviewer session; read the complete task file and `AGENTS.md`, confirmed REPLAY-02, XML-02, and CLI-02 are completed, and inventoried the pre-existing dirty/untracked work before review. No nested agents. Coordinator final task-file review remains after this gate.

Review correction FINAL-02-C1 (P2, acceptance-blocking for CLI-02 AC3): the CLI regression table skipped literal variants of HCL evaluation/type errors. Removing that exclusion made `go test -count=1 -run '^TestRoutesCLI/HCL' ./cmd/s3proxy` fail in `HCL_evaluated_key`: an attribute's literal/interpolated duplicate object key reached real CLI stderr. `decodeDiagnostics` protected expressions containing `env()` but retained this value-bearing HCL Detail for literal expressions. Deduplicated against prior CONFIG-02: that task explicitly covered environment-derived HCL diagnostics; the broader literal-value CLI acceptance here requires this correction. The shared diagnostic boundary now replaces duplicate-key Detail with a fixed uniqueness cause, retaining category, original location, and block/field identity; the existing env rule and ordinary numeric-type detail remain intact. The focused real-CLI regression is retained. This is a bounded correction under requirement 2, not an independent scope addition; final verification is pending below.

Review correction FINAL-02-C2 (P2, verification blocker): the first uncached `GOFLAGS=-count=1 make test-race` failed only `TestTransformBounds/exact_input_limit` (6.61 seconds under concurrent race instrumentation exceeded the production five-second transformation deadline). The earlier cached race run and fresh non-race boundary run passed. The size-bound fixture used the public timeout accidentally coupling two independent limits. Its call now uses the existing internal `transform` timeout parameter with 30 seconds, isolating exact/over-input, element, and output bounds from race overhead. Production `TransformTimeout` remains five seconds; dedicated cancellation/deadline and real-HTTP target-timeout tests remain enabled. This is a focused test correction required for FINAL-02's uncached race gate, not a protocol or resource-policy change; final verification is pending below.

Kind: investigation

Priority: P1 — independent completion gate for all changes.

Suggested agent: fresh integration reviewer who implemented none of the above.

Dependencies: REPLAY-02, XML-02, CLI-02.

Primary ownership: this task document; minimal corrective changes only when required by preceding acceptance criteria.

Finding: resource ownership, parsing invariants, and successful CLI output confidentiality need independent verification beyond implementer test reports.

References: all tasks and completion evidence above; repository `AGENTS.md`.

Requirements:

1. Inspect actual code/tests/docs against every acceptance criterion and recorded measurements. Check alternate replay readers/error exits, fragmented XML rejection paths, and command success/failure output boundaries.
2. Fix acceptance-blocking defects with focused regression evidence. Record independent new findings explicitly and deduplicate them rather than silently expanding scope.
3. Run `make vet test`, `make test-race`, `make build`, and verify `git diff --check` for this session's edited paths. Distinguish pre-existing diff issues if any.
4. Confirm `CHANGELOG.md` hash unchanged and no unrelated work removed. Review all task statuses/evidence and append a concise final acceptance matrix and material limitations.

Acceptance criteria:

- Each implementation task has delivered behavior, passing required checks, measurements where applicable, and accurate Completion evidence.
- Full vet/unit/race/build checks pass; final reviewer records commands/results.
- All four tasks completed with no hidden blocked criteria; task file and public docs match actual delivered contracts.

Verification: commands above and evidence review; live Docker/cloud tests not required because no remote protocol change is introduced.

Completion evidence (fresh independent FINAL-02 reviewer):

- Reviewed complete `AGENTS.md` and this plan before starting, confirmed all three dependencies completed, then marked FINAL-02 in progress. Inspected actual replay implementation, both replay test files and benchmark, auth/handler/dispatcher/backend ownership paths, XML parser/translation/encoding and fragmentation/bounds/backend/HTTP workflow tests, CLI command/DTO/subprocess tests, config loading/validation/diagnostics, resolver behavior, README and relevant design/operator/configuration docs. Completion summaries were cross-checked against those implementations and fresh executions.
- Findings: **FINAL-02-C1 fixed** (literal HCL duplicate-key confidentiality; regression failed before production correction and passed after); **FINAL-02-C2 fixed** (race-instrumented byte-bound test inadvertently enforced a wall-clock deadline). Details and original failing commands remain above. Both are required acceptance corrections. No separate independent finding, new follow-up task, unresolved acceptance blocker, or deferred criterion was identified. Prior CONFIG-02's environment-only HCL scope was explicitly checked when triaging C1.
- Session-edited paths: `cmd/s3proxy/routes_test.go`, `internal/config/diagnostics.go`, `internal/namespace/response_test.go`, `README.md`, `docs/design.md`, `website/docs/configuration.md`, and this task file. C1 adds a fixed duplicate-key diagnostic cause and enables both literal HCL subprocess cases; C2 changes only the bounds test's injected timeout. Documentation records the literal-key diagnostic rule. `make build` regenerated the requested ignored `dist/s3proxy` artifact. No replay or production XML code required correction.
- Preservation: initial and final `git status --short` have the same dirty/untracked path inventory; edits were narrow `apply_patch` changes to the paths above. No pre-existing work was removed, no whole-file restoration/reset/commit or nested agent was used, and no live stack was touched. Whitespace inspection was limited to session-edited paths; no claim is made about unrelated pre-existing diff whitespace.

Commands/results (all from `<repo-root>`; exit 0 unless a failure is explicitly recorded):

| Command                                                                                                                                                                                                                               | Result                                                                                                                                                                                                                                                                                                               |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `make vet test`                                                                                                                                                                                                                       | PASS before correction, after C1, and after C2; final vet plus all 17 unit packages passed (unchanged packages cached).                                                                                                                                                                                              |
| `make test-race`                                                                                                                                                                                                                      | Initial PASS, with some packages cached; superseded by the final uncached gate below.                                                                                                                                                                                                                                |
| `go test -count=1 -run '^TestRoutesCLI/HCL' ./cmd/s3proxy`                                                                                                                                                                            | FAIL before C1 (exit 1, literal evaluated-key sentinel disclosure); PASS after C1, including literal type conversion. Assertions never print the secret-bearing diagnostic.                                                                                                                                          |
| `go test -count=1 ./internal/replaybody ./internal/namespace ./internal/backend/s3 ./internal/httpapi ./cmd/s3proxy ./internal/config ./internal/app ./internal/router`                                                               | PASS, all eight packages uncached after C1; includes real successful/failed CLI subprocesses, `/dev/full`, occupied/non-bindable listeners and zero observed backend connections.                                                                                                                                    |
| `GOFLAGS=-count=1 make test-race`                                                                                                                                                                                                     | First run after C1 FAIL (exit 2 from make), only XML exact-input-bound fixture exceeded five seconds. After C2 PASS, all 17 packages uncached; namespace 10.111s, CLI 6.322s, backend 6.406s, HTTP 6.284s, replay 2.602s. No race reports. Covers and exceeds both implementation tasks' targeted race package sets. |
| `go test -race -count=1 -run 'TestTransformBounds\|TestTransformCancellationAndCloseError' ./internal/namespace`                                                                                                                      | PASS after C2, 9.783s; both byte/element/output bounds and independent deadline/close checks pass.                                                                                                                                                                                                                   |
| `make build`                                                                                                                                                                                                                          | PASS after C1 and at final gate; CGO-disabled host binary `dist/s3proxy`, version `079d2aa-dirty`.                                                                                                                                                                                                                   |
| `dist/s3proxy routes -c <repo-root>/tmp/s3proxy-cli02-example.hcl`                                                                                                                                                                    | PASS using the inspected pre-existing synthetic fixture; only the documented one-route JSON example was printed, with resolved `all`/`primary` labels and default `first`.                                                                                                                                           |
| `go test -run '^$' -bench BenchmarkEnsureReplay -benchmem ./internal/replaybody`                                                                                                                                                      | PASS, all ten fixtures; reviewer measurements below.                                                                                                                                                                                                                                                                 |
| `go test -run '^$' -bench '^BenchmarkParseResultFragmentedText$' -benchmem ./internal/namespace`                                                                                                                                      | PASS, all twelve fixtures; reviewer measurements below.                                                                                                                                                                                                                                                              |
| `gofmt -l cmd/s3proxy/routes_test.go internal/config/diagnostics.go internal/namespace/response_test.go`                                                                                                                              | PASS, no output; no formatting rewrite.                                                                                                                                                                                                                                                                              |
| `git diff --check -- README.md docs/design.md website/docs/configuration.md cmd/s3proxy/routes_test.go internal/config/diagnostics.go internal/namespace/response_test.go docs/tasks/20260926-163541-resource-and-operator-health.md` | PASS, no output.                                                                                                                                                                                                                                                                                                     |
| `git diff --no-index --check /dev/null cmd/s3proxy/routes_test.go`                                                                                                                                                                    | PASS, no output; explicitly checks this pre-existing untracked file.                                                                                                                                                                                                                                                 |
| `git diff --no-index --check /dev/null internal/config/diagnostics.go`                                                                                                                                                                | PASS, no output.                                                                                                                                                                                                                                                                                                     |
| `git diff --no-index --check /dev/null internal/namespace/response_test.go`                                                                                                                                                           | PASS, no output.                                                                                                                                                                                                                                                                                                     |
| `git diff --no-index --check /dev/null docs/tasks/20260926-163541-resource-and-operator-health.md`                                                                                                                                    | PASS, no output; repeated after final evidence edit.                                                                                                                                                                                                                                                                 |
| `sha256sum CHANGELOG.md`                                                                                                                                                                                                              | PASS: `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`, identical to the task's starting hash; checked at review and closeout.                                                                                                                                                                     |

Independent measurement check: same benchmark commands/fixtures, linux/amd64, Intel Core i9-13950HX, suffix `-32`. Fixture generation is outside timing; replay reads actually use one-byte/irregular sources; XML fixtures parse and verify exact decoded text. Reviewer results agree with recorded after-allocation counts. Historical before runs were reviewed for fixture/comparison consistency, not reconstructed by replacing the dirty worktree. Timing remains observational, with no threshold assertions.

| Reviewer fixture      |       B/op | allocs/op |      ns/op |
| --------------------- | ---------: | --------: | ---------: |
| Replay Known/1KiB     |      2,080 |        16 |      5,227 |
| Replay Known/1MiB     |  1,049,644 |        16 |  1,401,047 |
| Replay Known/Max      | 33,555,494 |        16 | 11,868,752 |
| Replay Unknown/1KiB   |     34,872 |        18 |     38,765 |
| Replay Unknown/1MiB   |  1,084,046 |        54 |  1,430,626 |
| Replay Unknown/Max    | 33,647,638 |     1,051 | 18,401,278 |
| Replay OneByte/1KiB   |     34,952 |        19 |    102,957 |
| Replay OneByte/64KiB  |     99,512 |        21 |  3,584,225 |
| Replay OneByte/1MiB   |  1,084,120 |        55 | 62,192,787 |
| Replay Irregular/1MiB |  1,084,129 |        55 |  1,460,930 |
| XML Plain/1KiB        |      6,200 |        80 |     34,090 |
| XML Plain/8KiB        |     27,704 |        83 |    188,648 |
| XML Plain/64KiB       |    199,742 |        86 |  2,080,168 |
| XML Comment/1KiB      |     12,696 |       339 |    155,256 |
| XML Comment/8KiB      |     86,684 |     2,137 |    988,178 |
| XML Comment/64KiB     |    681,894 |    16,480 |  8,858,581 |
| XML CDATA/1KiB        |      9,624 |       211 |     62,821 |
| XML CDATA/8KiB        |     62,106 |     1,113 |    402,605 |
| XML CDATA/64KiB       |    485,287 |     8,288 |  3,134,044 |
| XML Mixed/1KiB        |     12,696 |       339 |    153,043 |
| XML Mixed/8KiB        |     86,682 |     2,137 |  1,024,222 |
| XML Mixed/64KiB       |    681,931 |    16,480 |  7,239,869 |

Final acceptance matrix (AC numbers follow each task's listed criteria):

| Criterion                                              | Independent acceptance evidence                                                                                                                                                                                                                                                                                                                                                                                  | Result |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| REPLAY-02 AC1: reconstruction/concurrent replay/bounds | One-byte and irregular multi-chunk tests assert exact bytes, ceil(N/32768) chunk count, exact N capacity and bounded descriptor capacity; eight independent readers, partial reset, acquired-reader survival and close ownership pass. Scratch is not retained.                                                                                                                                                  | PASS   |
| REPLAY-02 AC2: limits/errors/reservations              | Exact request/aggregate limits, overflow, partial exhaustion preserving another reservation, data+EOF/error, no progress, cancellation and close-error paths inspected and run. Reserve-before-retain and release-on-error are preserved; auth/handler/dispatch/backend paths retain caller ownership and 413/503 mapping. Known oversize remains caller-owned before reading; existing GetBody remains a no-op. | PASS   |
| REPLAY-02 AC3: allocations/control behavior            | Fresh one-byte 1 MiB is 55 allocations / 1,084,120 B, versus historical 1,048,631 / 129,002,653; same allocation count as irregular reads. Known/ordinary unknown controls pass.                                                                                                                                                                                                                                 | PASS   |
| XML-02 AC1: equivalent text semantics                  | Plain/entity/comment/CDATA/mixed Unicode/metacharacter fixtures yield byte-identical virtual output; separate leaves, empty text, owner/checksum metadata and opaque tokens pass. Decoder nesting checks ensure the parser-local builder cannot mix valid nested leaves.                                                                                                                                         | PASS   |
| XML-02 AC2: linear allocation                          | Fresh 8→64 KiB increases allocated bytes about 7.87× (comment/mixed) and 7.81× (CDATA), for 8× text; no timing assertion. Builder finalizes and resets once per leaf.                                                                                                                                                                                                                                            | PASS   |
| XML-02 AC3: fail-closed/bounds/workflows               | Fragmented malformed, foreign/reset namespace, duplicate/nested/container/trailing content and sibling-key/prefix tests return no partial output and close once. Input/output/element/depth checks, cancellation and close errors inspected; bounds/deadline tests pass after C2. Signed list-then-get/atomic failure workflows and ordinary streaming/upstream-error paths pass.                                | PASS   |
| CLI-02 AC1: deterministic resolved topology            | Exact-schema subprocess assertions cover four routes, all parser/read modes, nonalphabetical route/operation/destination order, default first, bare/qualified/env-prefixed refs and repeated byte-identical runs. Built artifact matches public example.                                                                                                                                                         | PASS   |
| CLI-02 AC2: filters/arguments/failures                 | Explicit case-sensitive operations, omitted/empty/wildcard/duplicate rejection, empty topology, missing path/args, invalid config and short/failed writes pass. Config fully validates before DTO emission; config/default/resolver semantics agree.                                                                                                                                                             | PASS   |
| CLI-02 AC3: real output secrecy/offline behavior       | Private DTO emits declaration identities and validated vocabulary only. Full real CLI literal/env success/failure table passes, now including literal HCL duplicate-key/type failures after C1; stdout is empty for invalid config, stderr safe. Occupied/non-bindable listeners and zero counted backend connections confirm offline behavior; `/dev/full` fails safely.                                        | PASS   |
| CLI-02 AC4: DTO/docs boundary                          | No Runtime/Route serialization, map traversal output, reference-prefix output or app construction; fields copied explicitly from validated declarations. README/design/operator schema, public-label/filename boundary and updated diagnostic docs agree.                                                                                                                                                        | PASS   |
| FINAL-02 AC1: delivered behavior/evidence              | All ten preceding criteria checked against actual source/tests/docs and fresh measurements; historical completion notes retained, C1/C2 corrections explicitly supersede the incomplete literal-error coverage/timing assumption.                                                                                                                                                                                | PASS   |
| FINAL-02 AC2: full checks                              | Final vet/unit, uncached race, build, scoped formatting/whitespace and required changelog digest pass.                                                                                                                                                                                                                                                                                                           | PASS   |
| FINAL-02 AC3: statuses/contracts                       | REPLAY-02 completed; XML-02 completed; CLI-02 completed; FINAL-02 completed. No hidden blocked/pending/deferred task criterion. Coordinator final task-file review is still the next handoff.                                                                                                                                                                                                                    | PASS   |

Material limitations: replay accounting is a payload budget, not RSS; per-active-buffering-operation scratch, metadata, allocator overhead and already-acquired reader lifetime remain as documented. Cancellation of blocked sources relies on their Close contract. Listing bounds are per response, not aggregate process memory, and no pagination backend affinity is added. Race overhead is isolated only in the byte-bound test; the production five-second deadline remains unchanged. CLI labels and diagnostic filenames are public, finite vocabulary may originate in env, output device failures can leave partial bytes, and topology does not simulate routing/authorization or prove backend health. Historical before-benchmark timings were not independently reproduced. Live Docker/cloud/provider conformance, deployment load, dependency and release-artifact audits were outside this local review; no live verification is claimed. Coordinator closeout remains outstanding as requested, not an acceptance blocker for this independent gate.

## Definition Of Done

All four tasks completed by separate sequential agents; completion evidence includes required verification and measured performance results. Coordinator reviews the final task record after independent integration review. Existing working-tree work is preserved and `CHANGELOG.md` stays byte-identical.

## Coordinator Closeout

Continuation: [Request interpretation and probe boundary follow-up](20260926-170047-request-boundary-health.md) tracks the user's next review phase. This completed phase's statuses/evidence remain historical and unchanged.

- Reviewed the entire final task record, acceptance matrix, verification failures/corrections, measurements, final CLI/report boundary, diagnostic correction, and working-tree inventory after the independent gate. All four tasks are `completed` with Completion evidence; there are no unresolved blockers or acceptance criteria. Earlier notes saying later tasks or coordinator review were pending are historical handoffs, superseded by this closeout.
- Confirmed sequential isolated execution: REPLAY-02 session `ses_f1febca49ffe94IQtKCcOtewxj`; XML-02 session `ses_f1fe7ca9bffeEo49ZaIquyxz03`; CLI-02 session `ses_f1fe4f133ffeSWrTFyA0vcj4MF`; independent FINAL-02 session `ses_f1fe10151ffejaPlcGlKHtIYJJ`. Each finished before the next began; no implementation session was reused.
- Final gate evidence is complete: `make vet test`, `GOFLAGS=-count=1 make test-race`, `make build`, targeted regressions, benchmark suites, and scoped whitespace checks pass. The reviewer corrected and verified both discovered acceptance blockers before completing FINAL-02.
- Coordinator rechecked `sha256sum CHANGELOG.md`: `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`, unchanged. `git status --short` retains the pre-existing work inventory plus this plan's additions; no deletion is reported. No commits or live-stack operations were performed.
- Remaining product decisions and analysis limitations remain as documented above; no live cloud/Docker, deployment-load, dependency, or release audit is claimed. This objective and its requested final task-file review are complete.
