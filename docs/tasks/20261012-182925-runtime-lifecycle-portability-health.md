# Runtime Diagnostic, Replay Lifecycle, And Routing Portability Follow-Up

Created: 2026-10-12 18:29:25 (local)

Overall status: completed

## Objective And Scope

Continue the completed gateway review with three bounded outcomes: confidential
runtime rewrite diagnostics, correct replay-resource ownership after explicit
release, and consistent read placement across supported CPU architectures.
These support the stateless gateway's existing business workflows: private tenant
namespaces, bounded upload replay, and predictable replica selection.

Predecessor: `docs/tasks/20261012-162534-gateway-workflow-health.md` is completed.
All its uncommitted code, tests, and docs are intentional work to preserve. The
pre-existing `.github/actions/setup-tools/action.yml` edit also remains user-owned.

## Analysis And Baseline

- Inspected the current worktree; router/hash selection and tests; rewrite/template
  execution and HTTP logging/preflight; replay installation/release and lifecycle
  tests; release architectures; relevant operator/design docs and earlier task
  findings. Also inspected SDK signed-header handling; this plan makes no claim
  of a new authentication bypass from that inspection.
- Plan-creation baseline: `go test ./internal/rewrite ./internal/replaybody ./internal/router`
  passed before changes (cached). `CGO_ENABLED=0 GOARCH=386 go test ./internal/router`
  passed, establishing local 32-bit test execution. The then-existing hash tests
  checked same-process determinism/range, not a fixed cross-architecture mapping.
- An independent Python SHA-256 calculation gives `bucket/key3` a first-word hash
  of `ca7c2b99`: unsigned modulo three is 0, whereas absolute signed-32-bit modulo
  three is 1. `tenant-2/` gives `f72858a6`, with corresponding indices 1 and 0.
  These were analysis vectors at plan creation; subsequent Go regression results
  are recorded in ROUTEHASH-01 and FINAL-08.
- Earlier `docs/tasks/20260823-113640-codebase-health-follow-up.md` RESOURCE-01
  established immutable replay readers and cancellation safety, but its tests
  cancel the context rather than checking released resources with a live context.
  Its OBS-01 and the predecessor AUTHLOG-01 cover other logging boundaries.
  `docs/tasks/20260926-155056-final-review-follow-ups.md` CONFIG-02 protects
  template compilation errors; the runtime execution path is distinct.
- At plan creation, full checks, new regressions, and the final build were pending;
  their completed results are recorded in implementation and FINAL-08 evidence below.
  This is a focused follow-up, not another exhaustive audit of releases, cloud
  providers, dependencies, or all deferred product features. No live stack ran.

## Execution And Completion Rules

- Run fresh isolated sub-agents strictly sequentially:
  REWRITELOG-01 → REPLAY-LIFE-01 → ROUTEHASH-01 → FINAL-08.
  The final reviewer must be independent of all implementers.
- Follow `AGENTS.md` and the task-as-you-go workflow. Use `apply_patch`, preserve
  existing work, and format only owned files. Do not edit `CHANGELOG.md` or `.env`,
  commit, or start other agents from an assigned task.
- P2 denotes a confirmed privacy, resource-ownership, or deployment-correctness
  defect here. None is described as demonstrated production data exfiltration.
- All verification commands run from `<repo-root>`. Use synthetic credentials and
  repo-relative evidence. Do not record local absolute paths or private values.
- Each implementer adds meaningful regression-first coverage where feasible, runs
  its focused commands and `make vet test`, then updates its own status and concise
  `Completion evidence`. Include changed paths, exact commands, observed results,
  and material limitations; avoid exhaustive case transcripts.
- Shared design/operator docs are edited sequentially. Add necessary scope changes
  to requirements before implementation; independent work needs a fully specified
  follow-up rather than an undocumented expansion. A failed/unavailable required
  check leaves the task blocked with its prerequisite.
- Completion requires all four statuses/evidence, an independent acceptance matrix,
  required checks passing, and coordinator audit. User-visible corrections belong
  in product docs and the release-facing notes here, not `CHANGELOG.md`.

### Task REWRITELOG-01: Make Runtime Template Failures Value-Free

Status: completed

Kind: defect

Priority: P2 — runtime template failure diagnostics can expose private configuration
content despite the existing confidential configuration/logging contract.

Suggested agent: rewrite diagnostic implementer (fresh session)

Dependencies: none

Primary ownership:

- `internal/rewrite/apply.go` and focused diagnostic tests
- HTTP/app workflow tests for real template execution and captured logs
- `website/docs/operations.md`, `docs/design.md`

Finding:

`Apply` wraps the error returned by `CompiledTemplate.Execute` with `%w`. Go's
template execution errors include the failing action and can include literal
template text or capture names. `ServeHTTP` logs that raw error on rewrite failure.
An environment-derived template can therefore validate successfully but expose
its content when a request triggers a missing capture or an evaluation error.
`TestApply_KeyTemplateMissingCaptureFails` only requires a non-nil error.

References:

- `internal/rewrite/apply.go`: `engine.Apply`, template `Execute` error path
- `internal/httpapi/handler.go`: rewrite preflight and `rewrite failed` logging
- `internal/config/load.go`: template compilation with `missingkey=error`
- `internal/rewrite/apply_test.go`: `TestApply_KeyTemplateMissingCaptureFails`
- `internal/config/diagnostics.go`: existing safe compilation categories

Requirements:

1. Replace value-bearing execution errors at the rewrite boundary with a stable,
   useful, value-free cause. Do not retain raw template errors in an unwrap chain
   or depend on string-based redaction of secret values.
2. Preserve the 500/internal-error mapping for execution failure, 400 invalid-shape
   mapping, route/request correlation metadata, and preflight of all rewrites before
   any route dispatch. Valid templates and resulting bucket/key values are unchanged.
3. Cover literal/env template failures and direct rewriter diagnostics, plus actual
   HTTP requests through config compilation and real rewriting. Include missing
   captures and a failing action containing private literal text.
4. Keep HeadBucket's template-independent behavior and the existing restricted
   listing-template contract. Explain safe troubleshooting in operator/design docs.

Acceptance criteria:

- Returned errors and every formatted/unwrap view omit sentinel template values;
  logs/XML also omit them while retaining a fixed execution category and correlation.
- Regression-first coverage reproduces the leak. Validly authenticated/auth-none
  HTTP failures produce 500 with zero backend calls/object changes, including a
  failing later continued route. Avoid changing authentication/payload precedence.
- Existing valid template, namespace listing, and invalid-rewrite tests pass.
- Focused tests and `make vet test` pass; updated docs match behavior.

Verification: `go test ./internal/rewrite ./internal/httpapi ./internal/app ./internal/config`
and `make vet test`.

Completion evidence:

- Changed: `internal/rewrite/apply.go`, `internal/rewrite/diagnostics_test.go`,
  `internal/app/rewrite_diagnostics_test.go`, `docs/design.md`,
  `website/docs/operations.md`, and this document's REWRITELOG-01 evidence/status
  and release-facing note. The historical finding is preserved.
- Regression first: `go test -count=1 ./internal/rewrite ./internal/app -run '^(TestApply_TemplateExecutionDiagnosticsValueFree|TestBuild_RewriteExecutionDiagnosticsHTTP)$'`
  failed before production edits: eight direct object-method cases retained raw
  template causes/private input, and all 24 HTTP cases exposed input in logs.
  Existing 500/zero-backend/object-state assertions passed; assertions printed
  no sentinel values.
- `Apply` now returns only `key_template execution failed`, discarding the raw
  execution error. Direct coverage checks ordinary/detailed/quoted formatting,
  caller wrapping, `errors.As`, every unwrap view, zero partial results, distinct
  invalid-result classification, and HeadBucket bypass for both failing templates.
- HTTP coverage loads literal and `env()` templates through `app.Build`, exercising
  missing captures and a failing action with a private literal under auth-none,
  header SigV4, and presigning. Single/later continued-route failures retain generic
  500 XML, the fixed JSON cause, route/request/completion metadata, zero backend
  calls, and unchanged seeded objects. Auth-none reads no body; signed requests
  still verify their explicit payload hashes before rewriting.
- `go test ./internal/rewrite ./internal/httpapi ./internal/app ./internal/config`
  passed before edits (cached) and after the fix (fresh). Existing controls include
  `TestApply_KeyTemplate`, `TestListingNamespaceMatchesObjectRewrite`,
  `TestListingRejectsUnsafeMappingsAndHeadIgnoresKeyRules`, and
  `TestHandler_InvalidRewriteDoesNotDispatch`; authentication/payload suites pass.
- `make vet test` passed repository-wide after the fix.
- Operator/design docs explain safe troubleshooting and the unchanged 500/400,
  HeadBucket, and restricted listing contracts; the verified correction is noted
  below. `gofmt -d internal/rewrite/apply.go internal/rewrite/diagnostics_test.go internal/app/rewrite_diagnostics_test.go`,
  `git diff --check`, and `git diff --exit-code -- CHANGELOG.md` passed without output.
- Limits: local HTTP fixtures only; no live cloud/sandbox checks. No blockers or
  independent follow-ups found. Unrelated tracked diff digest matches the starting
  worktree; prior dirty/untracked work and other task statuses were preserved.

### Task REPLAY-LIFE-01: End Replay Cancellation Watching On Explicit Release

Status: completed

Kind: defect

Priority: P2 — released payloads and watcher goroutines remain live until context
cancellation, undermining reusable replay ownership with long-lived contexts.

Suggested agent: Go resource-lifecycle implementer (fresh session)

Dependencies: none

Primary ownership:

- `internal/replaybody/replaybody.go` and lifecycle/race regression tests
- Existing direct-backend/dispatch tests if needed to demonstrate composition
- `docs/design.md` replay ownership rationale

Finding:

For each installed nonempty replay with a cancelable context, `install` starts a
goroutine that waits only on `ctx.Done()` and holds `payload`. `Release` clears the
request and returns budget accounting, but `payload.release` does not notify that
watcher. Repeated calls using a context that stays alive retain goroutines and
their payload storage even after explicit release. Ordinary net/http contexts
cancel when handlers return, so this finding concerns the reusable direct-call
lifecycle rather than claiming all HTTP requests leak indefinitely. Existing race
tests cancel the context and therefore miss release-before-cancellation retention.

References:

- `internal/replaybody/replaybody.go`: `install`, `replayPayload.release`, `Release`,
  `replayReadCloser.Close`, `Reset`
- `internal/replaybody/replaybody_test.go`:
  `TestConcurrentGetBodyCancelBodyCloseAndRelease`,
  `TestReleaseIsIdempotentAndAcquiredReadersRemainValid`
- `internal/backend/s3/client.go`: `prepareSourceBody` and direct caller ownership

Requirements:

1. Stop/detach cancellation watching when replay ownership is explicitly released
   or the owning body is closed, without waiting for the request context to end.
   Keep registration and release race-safe if cancellation happens immediately.
2. Preserve once-only budget release, original-body closure, Reset/Release behavior,
   known/chunked replay bytes, and valid already-acquired readers after release.
   Do not mutate shared payload bytes or cancel the caller's context.
3. Add a meaningful bounded regression for repeated Ensure/release with a still-live
   context. Observe watcher/resource lifetime, not just the already-passing budget
   counter. Clean up all contexts even when the regression fails.
4. Cover body-close ownership, reset-then-release, and concurrent GetBody/release/
   cancellation. Any saved reader/GetBody closure legitimately retains its storage;
   document this distinction from unintended watcher retention.

Acceptance criteria:

- Regression-first reproduces retained watchers with a live context; after the fix,
  repeated released replays do not accumulate them and the context remains live.
- Owning close/release returns budget once; cancellation remains effective; acquired
  readers preserve exact bytes. No data race or new goroutine leak appears.
- Existing per-request/aggregate/no-progress/short-read and HTTP replay behavior
  remains correct. The fix stays encapsulated in replay lifecycle ownership.
- Focused/race tests and `make vet test` pass with concise lifecycle evidence.

Verification: `go test ./internal/replaybody ./internal/auth ./internal/dispatch ./internal/backend/s3 ./internal/httpapi`,
`go test -race ./internal/replaybody ./internal/dispatch`, and `make vet test`.

Completion evidence:

- Changed: `internal/replaybody/replaybody.go`,
  `internal/replaybody/replaybody_test.go`, `internal/replaybody/lifecycle_test.go`,
  `docs/design.md`, and this task's status/evidence and release-facing note. A
  notification initialized before watcher startup is closed by once-only release;
  the watcher can exit without canceling the caller or mutating shared bytes.
- Regression first: `go test -count=1 -v ./internal/replaybody -run '^TestReleasedReplayLifetimeWithLiveContext$'`
  failed before production edits in all six known/unknown-length release,
  owning-close, and reset-release cases. Each batch retained 32 payloads and
  watchers, plus 32 contiguous or 96 chunk allocations, despite correct budget
  accounting and live contexts.
- After the fix, all six cases observed zero reachable payloads/backing allocations
  and zero watchers (baseline zero), with contexts still live. Each case releases
  32 nonempty 65,673-byte replays; standard-library weak references, stack counts,
  and bounded five-second GC/lifecycle polling observe resources beyond accounting.
  Context cleanup is registered before assertions, including failure paths.
- `TestEnsureCancellationAtReplayInstall` cancels during original-body closure,
  before watcher installation. The concurrent GetBody/cancel/owning-close/release
  test covers 100 rounds per storage layout with exact bytes, pre-acquired readers,
  saved closures used after release, once-only source closure, and an independent
  reservation that detects double release. Live-context cases also verify rewind,
  non-owning reset-reader closure, and partial-reader validity. The idempotence
  fixture now actually exercises replay installation rather than an existing GetBody.
- `go test ./internal/replaybody ./internal/auth ./internal/dispatch ./internal/backend/s3 ./internal/httpapi`
  passed before edits (cached) and after the fix (fresh), including existing limits,
  aggregate reservations, no-progress/short-read, auth, backend, and HTTP controls.
- `go test -race ./internal/replaybody ./internal/dispatch` and `make vet test`
  passed. The new lifetime/install tests and strengthened concurrent-reader tests
  also passed in a fresh focused run after the fix.
- `docs/design.md` explains notification ordering, reset ownership, and intentional
  retention by readers/saved closures; the correction is noted below.
  `gofmt -d internal/replaybody/replaybody.go internal/replaybody/replaybody_test.go internal/replaybody/lifecycle_test.go`,
  `git diff --check`, and `git diff --exit-code -- CHANGELOG.md .env` passed without
  output. Unrelated tracked-diff and prior untracked-file hashes match the baseline;
  shared-doc edits preserve previous-phase and REWRITELOG-01 content.
- Limits: local lifecycle/HTTP fixtures and bounded GC observations, not an RSS or
  live-cloud/sandbox measurement. Saved readers/closures legitimately retain their
  bytes; ordinary net/http contexts already cancel after handlers. No blockers or
  independent follow-ups found.

### Task ROUTEHASH-01: Make Replica Hash Selection Architecture-Independent

Status: completed

Kind: defect

Priority: P2 — mixed 32-/64-bit deployments can send the same read/list request to
different replicas despite the deterministic hash preference.

Suggested agent: routing portability implementer (fresh session)

Dependencies: none

Primary ownership:

- `internal/router/resolve.go`, hash/resolver regression tests
- `website/docs/configuration.md`, `README.md`, `docs/design.md`

Finding:

`hashIndex` accumulates the first four SHA-256 bytes in a native `int`, negating
negative values. On 64-bit builds this is the unsigned 32-bit value; on 32-bit
builds a set high bit becomes negative and is folded to a different value. The
signed minimum also cannot be made positive by negation. Releases include 386
and ARM alongside 64-bit targets. `TestHashIndex` checks only determinism/range
for one vector whose high bit is clear, so the existing 386 suite misses this inconsistency.

References:

- `internal/router/resolve.go`: `hashIndex`, `selectEffectiveRead`
- `internal/router/resolve_test.go`: `TestHashIndex`, `TestResolve_EffectiveRead`
- `go-release.json`: linux/windows 386 and linux ARM targets
- Analysis vectors in this document's baseline section

Requirements:

1. Define selection using the unsigned big-endian first 32 bits of
   SHA-256(bucket + "/" + raw key), modulo destination count, using arithmetic
   that cannot become negative or depend on native int width.
2. Preserve current 64-bit mappings, original pre-rewrite bucket/key inputs,
   destination order, empty-key listing behavior, and one/no-destination guards.
3. Add fixed independently derived vectors with both high-bit states and multiple
   destination counts; exercise actual resolver selection for object/list reads.
   Do not brute-force a SHA-256 preimage to test arithmetic boundaries.
4. Run the same tests natively and as a real executable 386 test binary. Document
   the intentional correction to some 32-bit read selections and consistent rollout/
   pagination implications; do not claim backend-pinned cursors or move object data.

Acceptance criteria:

- New regressions fail on the old 386 implementation and pass natively/386 after
  the fix, with identical explicit expected destination indices.
- All current 64-bit mappings are preserved by the fixed unsigned formula; index
  arithmetic is always in range for positive destination counts on both widths.
- Resolver coverage preserves read/list identity, destination ordering, and ordinary
  write selection. Product docs state the corrected portability contract.
- Focused tests, required 386 execution, and `make vet test` pass.

Verification: `go test ./internal/router ./internal/httpapi`,
`CGO_ENABLED=0 GOARCH=386 go test ./internal/router`, and `make vet test`.

Completion evidence:

- Changed: `internal/router/resolve.go`, `internal/router/hash_test.go`,
  `README.md`, `website/docs/configuration.md`, `docs/design.md`, and this task's
  status/evidence and release-facing note. The first hash word is decoded as
  unsigned big-endian `uint32`, with `uint64` modulo before conversion to `int`;
  the existing 64-bit mapping and hash inputs are preserved.
- Regression first: before production edits,
  `go test ./internal/router ./internal/httpapi` passed on linux/amd64.
  `CGO_ENABLED=0 GOARCH=386 go test ./internal/router` executed and failed 15
  fixed-vector and 28 resolver cases. Low-bit, guard, and write controls passed;
  this reproduced selection differences rather than a cross-compilation failure.
- Independent Python `hashlib.sha256` / big-endian `int.from_bytes` vectors fix
  the expected indices. With three destinations: `bucket/key1` (`2669e9f6`) → 1,
  `bucket/key3` (`ca7c2b99`) → 0, `tenant-2/` (`f72858a6`) → 1, and
  `bucket/a%252Fb` (`ffc70e24`) → 0. Old 386 selected 1, 0, and 1 respectively
  for the latter three high-bit vectors.
- `TestHashIndexFixedVectors` covers six high/low-bit object, empty-list-key,
  and escaped-key vectors at counts 2, 3, 5, 7, 257, and 2147483647.
  `TestHashIndexWideDestinationCounts` checks 2^31, 2^32, and 2^63−1 directly,
  without destination allocations; these three unrepresentable counts skip on 386. Minimum-int/negative/zero/one counts and no/sole-destination guards are
  covered. No special SHA preimage or production test-only abstraction was added.
- `TestResolve_HashReadSelection` covers GET/HEAD objects, ListObjectsV2, and
  HeadBucket through real path-style/virtual-hosted parsing, two destination
  orders, and subsequent bucket/prefix rewriting. Raw escape case, encoded versus
  literal separators, percent/Unicode/repeated slashes, empty listing keys, and
  filter/token-independent listing selection retain fixed expected indices.
  `TestResolve_HashWritesUseFirstDestination` preserves PUT/DELETE first-target
  selection and the full destination list for both dispatch modes.
- After the fix, `go test ./internal/router ./internal/httpapi`,
  `CGO_ENABLED=0 GOARCH=386 go test ./internal/router`, and `make vet test` all
  passed. Both focused router runs executed freshly with identical expected
  mappings; required 386 verification ran the tests, not just compilation.
- README/configuration/design and the note below document the 32-bit correction,
  consistent mixed-version rollout, pagination restart implications, opaque
  unpinned tokens, and unchanged write/storage semantics.
  `gofmt -d internal/router/resolve.go internal/router/hash_test.go`,
  `git diff --check`, and `git diff --exit-code -- CHANGELOG.md .env` passed
  without output.
- Unrelated tracked-diff and prior untracked-file hashes match the starting
  worktree, including predecessor work and the user workflow edit; shared-doc
  edits preserve earlier task content. Limits: local amd64/386 execution only,
  with no ARM32 runtime, full release-platform matrix, or live cloud/sandbox run.
  No blockers or independent follow-ups found.

### Task FINAL-08: Independently Review Follow-Up Acceptance And Integration

Status: completed

Kind: investigation

Priority: P2 — independent completion gate for privacy, resource lifetime, and
cross-architecture routing corrections.

Suggested agent: independent Go integration reviewer (fresh session)

Dependencies: REWRITELOG-01, REPLAY-LIFE-01, ROUTEHASH-01

Primary ownership: this task's acceptance matrix/evidence and all phase changes as
reviewer; bounded acceptance corrections with regressions if necessary.

Finding: the three fixes affect existing contracts already hardened by earlier
phases. Independent verification must establish corrected behavior without losing
signature/preflight, immutable-reader, or replica-selection guarantees.

References: the task findings, requirements, and completion evidence above;
predecessor `docs/tasks/20261012-162534-gateway-workflow-health.md`.

Requirements:

1. Inspect all acceptance criteria against production control flow, tests, docs,
   and recorded regression-first evidence. Use a concise matrix covering each task's
   criteria; inspect untracked tests as well as tracked diffs.
2. Verify runtime template errors have no raw causes, released watchers detach while
   readers stay valid, and native/386 fixed-vector routing matches. Confirm earlier
   token/endpoint/header-boundary work remains intact.
3. Run `make vet test`, `make test-race`, `make build`, and
   `CGO_ENABLED=0 GOARCH=386 go test ./internal/router`; inspect the build parent
   directory first. Check whitespace and unchanged `CHANGELOG.md`/prior work.
4. Record and fix bounded acceptance defects with necessary rechecks. Independent
   scope additions require specified follow-up tasks before execution/sign-off.

Acceptance criteria:

- All three implementation tasks meet their acceptance criteria with reproducible
  evidence; no required check or follow-up is unresolved.
- Full vet/unit/race/build and focused native/386 verification pass.
- Implementation/docs/tests agree; all four task statuses and evidence are complete.
- `CHANGELOG.md`, `.env`, previous-phase work, and the original workflow edit are
  preserved; no unintended commit or artifact inclusion occurs.

Verification: the commands above, `git diff --check`,
`git diff --exit-code -- CHANGELOG.md`, and independent criterion review.

Acceptance matrix:

AC numbers follow each task's acceptance bullets in their original order. The
review traced production control flow and ownership, inspected assertions rather
than relying on summaries, and checked recorded regression-first results against
the fixtures. Those historical failing runs remain implementation evidence.

| Task / criterion                                                   | Result | Independent acceptance basis                                                                                                                                                                                                                                                                                                                                                                                                              |
| ------------------------------------------------------------------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| REWRITELOG-01 AC1 — value-free errors and telemetry                | Pass   | `Apply` returns a new fixed error without retaining `template.ExecError` or any cause. `internal/rewrite/diagnostics_test.go` checks all formatting/unwrap views, caller wrapping, and zero partial results; `internal/app/rewrite_diagnostics_test.go` checks actual JSON logs/XML and route/request/completion metadata.                                                                                                                |
| REWRITELOG-01 AC2 — authenticated preflight without side effects   | Pass   | The 24 app HTTP cases compile literal/env templates and exercise real auth-none/header/presign PUTs, including a failing later continued route: 500, zero backend calls, unchanged objects. Handler authorization and all-route rewrite/shape preflight precede dispatch-related replay and dispatch; signature/payload verification retains its earlier position.                                                                        |
| REWRITELOG-01 AC3 — valid templates, shape, listing, HeadBucket    | Pass   | Existing valid-template and namespace list-then-get controls pass. Invalid object shapes still map to 400; execution failures remain 500. HeadBucket bypasses even the new failing templates. Listing uses restricted AST prefix evaluation and fixed mapping/query errors, preserving its separate 400 contract.                                                                                                                         |
| REWRITELOG-01 AC4 — checks and troubleshooting docs                | Pass   | Rewrite/config/app/HTTP packages pass within full vet/unit/race verification. Design/operator docs accurately describe discarded raw causes, correlation, private inspection with synthetic requests, and the limits of offline template validation.                                                                                                                                                                                      |
| REPLAY-LIFE-01 AC1 — release with a live context                   | Pass   | The uncached race run observes zero reachable payloads/backing allocations via weak references and zero watcher stacks after each 32-replay batch in all six known/chunked release, owner-close, and reset-release cases; every context remains live. This tests resource lifetime beyond budget counters.                                                                                                                                |
| REPLAY-LIFE-01 AC2 — race safety, once-only release, valid readers | Pass   | The notification is initialized before watcher startup. `sync.Once` returns budget and closes it without waiting on the watcher, so immediate cancellation and owning-close/release cannot deadlock recursively or double-release. Install-time cancellation, 200 concurrent-reader rounds, saved closures, and pre-acquired/partial readers preserve exact bytes; the independent 17-byte reservation detects double accounting release. |
| REPLAY-LIFE-01 AC3 — ownership and composed replay behavior        | Pass   | Reset installs a non-owning reader; closing it leaves ownership intact, while Release explicitly releases its payload and clears request hooks. Source closure, known/unknown bytes, per-request/aggregate limits, short reads, and no-progress controls remain intact. Auth, direct backend, dispatch fan-out, and HTTP suites cover the shared budget/reader call paths.                                                                |
| REPLAY-LIFE-01 AC4 — regression quality and checks                 | Pass   | Full checks and fresh focused race tests pass. Helpers drop strong replay references before GC; payload/storage observations complement package-specific watcher counts. Bounded polling and cancellation cleanup registered before assertions make the original retention regression observable and clean up failing cases. Design docs distinguish accounting release from intentional reader/closure retention.                        |
| ROUTEHASH-01 AC1 — independent fixed native/386 mappings           | Pass   | Independent Python SHA-256 calculations agree with all six fixed tables and additional raw-key resolver expectations. Native router tests execute in the full race run; uncached 386 execution passes all 36 fixed-vector comparisons and 56 read-resolution cases. Recorded old-386 failures match the signed-folding defect.                                                                                                            |
| ROUTEHASH-01 AC2 — range and 64-bit compatibility                  | Pass   | The unsigned word is in [0, 2^32−1], exactly the old 64-bit accumulator. For positive native counts, uint64 modulo yields a representable result below the count. Counts at/below one are guarded; max-int32 and native-wide counts are covered. Only the three counts exceeding int32 skip on 386.                                                                                                                                       |
| ROUTEHASH-01 AC3 — resolver identity, ordering, writes             | Pass   | Resolver tests parse both addressing modes, preserve raw escapes/empty listing keys and pre-rewrite inputs, vary destination order independently of registry order, and ignore listing filters/tokens. PUT/DELETE retain the first target and full destination list under both dispatch modes; no/sole-destination guards pass. Dispatch still sends reads once and fans out only configured supported writes.                            |
| ROUTEHASH-01 AC4 — checks and rollout contract                     | Pass   | Required checks and actual 386 execution pass. README/configuration/design describe the 32-bit correction, unchanged 64-bit placement, equivalent replicas, consistent algorithm/order, and restarting affected pagination without pinned cursors or data movement. The API pagination summary was qualified to match that contract during review.                                                                                        |
| Predecessor regression guards — FINAL-08 requirement 2             | Pass   | Inspected preserved selector-before-auth-filter handling, value-free auth causes, offline endpoint validation, inbound token stripping, and selected-target session signing. Existing header/payload, endpoint/CLI, token-isolation/independent-signature, and catalog ownership tests pass in full checks. These remain predecessor work; its benchmark was not remeasured.                                                              |
| FINAL-08 AC1 — complete implementation acceptance                  | Pass   | All 12 implementation criteria above are satisfied. The bounded API documentation gap is corrected; no production acceptance defect, blocker, or independent follow-up remains.                                                                                                                                                                                                                                                           |
| FINAL-08 AC2 — required verification                               | Pass   | Full vet/unit/race, host build, native router coverage, actual uncached 386 router execution, and tracked/untracked whitespace checks pass as recorded below.                                                                                                                                                                                                                                                                             |
| FINAL-08 AC3 — implementation/docs/tests and task evidence         | Pass   | All four individual tasks now have completed statuses and evidence. Only the API summary and this document's stale baseline/release-facing wording needed clarification. The subsequent parent audit is recorded in Coordinator Closeout.                                                                                                                                                                                                 |
| FINAL-08 AC4 — preservation and evidence hygiene                   | Pass   | Protected-file and prior-untracked fingerprints match review entry; all other tracked diffs are unchanged except the reviewed API paragraph. The predecessor's API header documentation and original workflow `github-token` edit remain intact. Evidence uses repo-relative paths and synthetic fixtures; no staging, commits, or unintended artifact inclusion occurred.                                                                |

Completion evidence:

- Independent review read `AGENTS.md`, this entire task file and its predecessor,
  all intended tracked diffs, all 15 untracked test/benchmark files, and surrounding
  config/app, auth/handler, namespace/rewrite, replay/backend/dispatch, and request
  parsing/router paths. No agents were launched.
- Correction: `website/docs/api-reference.md` previously described hash pagination
  as stable for unchanged destinations without qualifying mixed old-32-bit and
  corrected algorithms. Its existing listing paragraph now requires a consistent
  algorithm and ordered destinations, explains affected-listing restart, and links
  the rollout guidance. Existing fixed-vector/resolver regressions substantiate
  the correction; the final diff and whitespace check pass. No code/test change
  was needed. This document adds acceptance evidence/status and clarifies historical
  baseline and verified release-facing wording without changing original findings.
- Required commands from `<repo-root>`, Go 1.27.1 linux/amd64:
  - `make vet test` — exit 0; vet clean, all 19 unit-test packages passed (cached).
  - `make test-race` — exit 0; all 19 packages passed with no race report;
    10 executed freshly, including app/rewrite/router, and nine reused cached results.
  - `ls -ld . dist` — inspected build parent directories before `make build`.
    `make build` — exit 0; CGO-disabled host binary built at `dist/s3proxy` with
    `-trimpath -buildvcs=false`. `git check-ignore -- dist/s3proxy` confirms it is ignored.
  - `CGO_ENABLED=0 GOARCH=386 go test ./internal/router` — exit 0 (cached);
    the fresh execution below closes that review-evidence gap.
  - `git diff --check` and `git diff --exit-code -- CHANGELOG.md` — exit 0,
    no output. Whitespace was rechecked after the API documentation correction.
- Targeted uncached checks addressed cached runtime evidence, rather than repeating
  the full suites:
  - `CGO_ENABLED=0 GOARCH=386 go test -count=1 -v ./internal/router` — exit 0,
    fresh executable test run (0.024s), including all 36 fixed-vector cases,
    56 read-resolution cases, write controls, and guards. The observed minimum-int
    guard is −2147483648; only the three unrepresentable wide destination counts
    skip. This is actual 386 execution, not cross-compilation alone.
  - `go test -race -count=1 -v ./internal/replaybody -run '^(TestReleasedReplayLifetimeWithLiveContext|TestEnsureCancellationAtReplayInstall|TestConcurrentGetBodyCancelBodyCloseAndRelease|TestReleaseIsIdempotentAndAcquiredReadersRemainValid)$'`
    — exit 0 (6.239s), no races. Each of six 32 × 65,673-byte batches reports
    zero reachable payloads/storage and zero watchers (baseline zero), with its
    context still live. Both install-cancellation cases and 100 concurrent rounds
    per storage layout pass, including exact post-release reader/closure bytes.
- Lifecycle assertion audit: `go doc weak.Pointer` confirms weak observations do
  not root the payload. The helper returns only weak references after its readers
  and saved closures leave scope; backing allocations are 65,673 bytes or chunks
  of 32,768/32,768/137 bytes, avoiding tiny-allocation batching. Stack counts select
  replay-install closures rather than all process goroutines, with complete stack
  capture and five-second GC polling. Deferred/registered cancellation precedes
  assertions, so failure paths also end live contexts. These are bounded runtime
  observations, not a claim of an RSS bound or universally guaranteed GC timing.
- Independent arithmetic command:
  `python3 -c 'from hashlib import sha256; cases = [("bucket", k) for k in ("key1", "key3", "", "a%2Fb", "a%252Fb", "a%2fb", "a/b", "%E9%9B%AA/%25//a%2Fb", "/a//b")] + [("tenant-2", "")]; counts = (2, 3, 5, 7, 257, 2**31-1); print("counts:", counts); [(print(b + "/" + k, sha256((b + "/" + k).encode()).hexdigest()[:8], [int.from_bytes(sha256((b + "/" + k).encode()).digest()[:4], "big") % n for n in counts])) for b, k in cases]; print("wide key3:", [int.from_bytes(sha256(b"bucket/key3").digest()[:4], "big") % n for n in (2**31, 2**32, 2**63-1)])'`
  — exit 0; fixed tables, the four additional raw-key resolver expectations, and
  wide-count results match `internal/router/hash_test.go`.
- Formatting/whitespace:
  `gofmt -d internal/rewrite/apply.go internal/rewrite/diagnostics_test.go internal/app/rewrite_diagnostics_test.go internal/replaybody/replaybody.go internal/replaybody/replaybody_test.go internal/replaybody/lifecycle_test.go internal/router/resolve.go internal/router/hash_test.go`
  — exit 0, no diff.
  `git ls-files --others --exclude-standard -z | xargs -0 -r -n 1 git diff --no-index --check -- /dev/null`
  — exit 0, no output for all 15 untracked test/benchmark files and both task files.
- Preservation: `git diff --binary -- . ':(exclude)website/docs/api-reference.md' | git hash-object --stdin`
  matches review entry. Untracked-file hashes match except this intentionally updated
  task file. `git hash-object -- .env CHANGELOG.md .github/actions/setup-tools/action.yml docs/tasks/20261012-162534-gateway-workflow-health.md`
  retains every starting fingerprint. The API diff preserves the predecessor's
  header-boundary addition; its only review change is the pagination qualification.
  `git diff --cached --exit-code` passes with no output. The file inventory contains
  the expected work only, with no private values or undesired files added to evidence.
- Limits: local HTTP fixtures and synthetic credentials, host/386 execution, and
  bounded Go lifecycle observations. No live cloud/Docker stack, ARM32 execution,
  full release-platform runtime matrix, or new catalog benchmark measurement ran.
  FINAL-08 has no blocker; the subsequent parent audit is recorded in Coordinator Closeout.

## Release-Facing Notes

Verified during implementation and independently accepted in FINAL-08: runtime
diagnostic confidentiality, explicit replay release, and portable hash placement.
External contract/rollout implications are below. `CHANGELOG.md` remains excluded.

- Runtime object-template failures now use the value-free
  `key_template execution failed` diagnostic with route/request correlation;
  private literal/environment-derived template contents and raw causes are
  discarded. `500 InternalError`, all-route rewrite preflight, and the separate
  `400 InvalidRequest` shape/listing contract are preserved. Troubleshoot using
  private configuration inspection and synthetic requests.
- Explicit replay release and owning-body closure now stop cancellation watchers
  even with long-lived direct-call contexts, including `Reset` then `Release`.
  Caller contexts remain live and budget returns once. Acquired readers and saved
  `GetBody` closures keep immutable bytes valid; retaining those references still
  retains their storage.
- `read_preference = "hash"` now uses the unsigned big-endian first SHA-256 word
  consistently on 32-/64-bit builds, preserving 64-bit mappings while correcting
  some older 386/ARM32 selections. Original pre-rewrite bucket/raw-key inputs,
  destination order, write dispatch, and stored objects retain their semantics.
  Keep the algorithm and ordered destination configuration consistent during
  rollout; backend changes can invalidate ongoing list tokens or yield
  inconsistent pages. Finish with stable routing or restart affected listings.
  Tokens remain opaque, without proxy-managed backend affinity.

## Deferrals And Limits

The product's stateful/dynamic features remain outside this follow-up: multipart,
CopyObject, merged/backend-pinned listing cursors, credential refresh, and hot
reload. No new maintainer decision blocks these three fixes. Live cloud/sandbox
tests and full multi-platform runtime testing are supplementary; local HTTP
fixtures, host checks, and actual 386 router execution are the required evidence.

## Coordinator Closeout

Completed: 2026-10-12 19:13:37 PDT (local audit)

- Read the entire task file after all four isolated sub-agent sessions completed.
  Confirmed four unique task IDs, four completed statuses, four completion-evidence
  sections, satisfied dependencies, and independent acceptance of all 12
  implementation criteria. No required check or follow-up remains open.
- Execution was sequential: REWRITELOG-01, REPLAY-LIFE-01, ROUTEHASH-01, then a
  fresh independent FINAL-08 reviewer. Reviewed production changes between agents
  and accepted the final runtime/native/386 verification and API-doc correction.
- Required vet/unit/race/build and 386 router checks passed. Fresh targeted race
  and 386 runs substantiate actual watcher cleanup and cross-architecture selection.
  Only task-document closeout wording changed after the independent review.
- Final worktree inventory preserves the preceding completed phase and original
  workflow edit. Independently confirmed `git diff --exit-code -- CHANGELOG.md`
  passes; no commits were made. Evidence and references are repo-scoped.
- Historical regressions, rollout implications, legitimate reader/closure retention,
  bounded GC evidence, and live-cloud/ARM32 limitations remain explicit. The
  document's completion requirements are satisfied.
