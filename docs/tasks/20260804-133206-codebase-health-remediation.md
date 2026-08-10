# Codebase Health Remediation

Created: 2026-08-04 13:32:06 local time

## Objective And Scope

Remediate confirmed correctness, security, resource-lifecycle, readability, and architectural-health gaps found in a read-only review of the Go proxy. This document is intended for independent sub-agents and includes ownership, dependencies, regression tests, and repository verification commands.

The highest-risk work is streaming correctness and the inbound-to-outbound trust boundary. Architectural changes follow behavioral fixes so refactoring cannot conceal regressions.

## Working Rules And Non-Goals

- Preserve unrelated worktree changes. At review time `.github/workflows/publish.yaml` was modified and is outside this plan.
- Add a failing regression test before each confirmed bug fix.
- Prefer the smallest shared enforcement point; do not perform a broad domain-model rewrite.
- Preserve escaped-path behavior and explicit outbound `Content-Length` signing requirements documented in `AGENTS.md`.
- Do not add multipart uploads, hot reload, database-backed auth, backend health polling, or merged listings.
- Do not treat configuration-origin SSRF as a remote vulnerability; configuration is currently a privileged trust boundary.
- Update this file's status and completion evidence as work progresses.
- Agents sharing `internal/httpapi/handler.go`, `internal/auth/verify.go`, `internal/backend/s3/client.go`, or `internal/config/validate.go` must run sequentially.

## Baseline Verification

Review baseline on 2026-08-04:

- `make vet test`: passed.
- Integration tests: not run; they require the live Docker sandbox.
- No existing files were present under `docs/tasks/`.
- Direct test gaps: `cmd/s3proxy`, `internal/app`, and `internal/replaybody` have no package tests.

After each task, run its targeted tests. After each wave, run `make vet test`. Use `make test-race` for fan-out, shared-state, or concurrency changes. Run `make sandbox-integration-up` only when the live sandbox is available.

## Priority Definitions

- P0: release-blocking security or data-correctness defect.
- P1: significant correctness, resource exhaustion, lifecycle, or operational risk.
- P2: maintainability, testability, observability, or bounded optimization.
- P3: optional optimization requiring measurement first.

## Execution Waves

| Wave | Tasks                        | Parallelization                                                                     |
| ---- | ---------------------------- | ----------------------------------------------------------------------------------- |
| 1    | STREAM-01, AUTH-01           | Parallel; separate primary files                                                    |
| 2    | ROUTE-01, HTTP-01, CONFIG-01 | Parallel except ROUTE-01 and HTTP-01 both touch handler tests and should coordinate |
| 3    | RESOURCE-01, RESOURCE-02     | Sequential; both affect response ownership                                          |
| 4    | APP-01, ARCH-01, OPS-01      | Parallel after earlier handler/auth contracts settle                                |
| 5    | PERF-01, TEST-01             | Parallel; PERF-01 is measurement-gated                                              |
| 6    | INTEGRATION-01               | Independent final reviewer only                                                     |

Recommended agent allocation:

| Agent                | Focus                  | Primary ownership                                  |
| -------------------- | ---------------------- | -------------------------------------------------- |
| Security agent       | AUTH-01                | `internal/auth`, authenticated header contract     |
| Streaming agent      | STREAM-01, RESOURCE-01 | backend timeout and response ownership             |
| Routing agent        | ROUTE-01               | operation classification and multi-route semantics |
| Configuration agent  | CONFIG-01              | cross-reference validation                         |
| Lifecycle agent      | APP-01                 | app/CLI ownership and tests                        |
| Architecture agent   | ARCH-01                | narrow interfaces and immutable snapshots          |
| Operations agent     | OPS-01                 | request and destination telemetry                  |
| Performance agent    | RESOURCE-02, PERF-01   | replay budget and measured optimizations           |
| Independent reviewer | INTEGRATION-01         | acceptance audit and full verification             |

## Wave 1: Trust And Streaming Correctness

### Task STREAM-01: Preserve Target Timeout Through Response Streaming

Status: completed 2026-08-09

Priority: P0

Suggested agent: Go HTTP streaming specialist

Dependencies: none

Primary ownership:

- `internal/backend/s3/client.go`
- `internal/backend/s3/client_test.go`
- focused response-copy tests in `internal/httpapi/handler_test.go`

Finding:

`client.Do` creates a target timeout context and defers its cancellation before returning a still-streaming response. The cancellation runs as soon as headers are returned, so configured target timeouts can cancel successful response bodies before the handler reads them. The handler then discards `io.Copy` errors, allowing a nominal success with a truncated body.

References:

- `internal/backend/s3/client.go:48-53`
- `internal/backend/s3/client.go:95-104`
- `internal/httpapi/handler.go:207-221`
- `internal/backend/s3/client_test.go:149-190`

Implementation requirements:

1. Keep the timeout active until the returned body reaches EOF or is closed; cancel immediately on request construction, signing, or transport errors.
2. Make response-body cancellation ownership explicit with a small `io.ReadCloser` wrapper rather than leaking cancel functions into callers.
3. Change response writing to detect and return/log copy errors while always closing the body.
4. Preserve streaming; do not buffer response bodies.

Acceptance criteria:

- A backend that flushes headers and then streams within the configured target timeout is read successfully after `Do` returns.
- A body stream that exceeds the target timeout terminates with a deadline error and closes resources.
- A body returning data followed by a sentinel error is closed and the error is observable in a handler test/log assertion.
- `go test ./internal/backend/s3 ./internal/httpapi` passes.

Completion evidence 2026-08-09:

- Added backend regression coverage for header-flushed responses that continue streaming after `Do` returns and for body reads that exceed the target timeout.
- Added handler coverage proving response body copy errors are logged and the body is closed.
- `go test ./internal/backend/s3 ./internal/httpapi`: passed.
- `make vet test`: passed.

### Task AUTH-01: Enforce The Authenticated Request Boundary

Status: completed 2026-08-09

Priority: P0

Suggested agent: SigV4/security specialist

Dependencies: none

Primary ownership:

- `internal/auth/verify.go`
- `internal/auth/auth.go`
- `internal/auth/auth_test.go`
- authenticated-header filtering seam used by `internal/backend/s3/client.go`

Finding:

Verification strips unsigned headers only from a signing clone. The original request retains every header, and the backend client forwards nearly all of them before signing with privileged destination credentials. A presigned URL holder or intermediary can therefore add unsigned S3 control headers such as ACL, tagging, metadata, storage class, and conditionals. Header-signed requests also accept a missing `X-Amz-Date`, and credential scope structure/date/service/terminator are not fully validated. Finally, a non-`UNSIGNED-PAYLOAD` hash is trusted as a signed string without checking the body bytes.

References:

- `internal/auth/verify.go:45-102`
- `internal/auth/verify.go:105-163`
- `internal/auth/verify.go:165-228`
- `internal/auth/verify.go:269-308`
- `internal/backend/s3/client.go:81-93`
- `internal/auth/auth_test.go:163-199`
- `internal/auth/auth_test.go:249-273`

Implementation requirements:

1. Return or attach an authenticated-header policy/result from verification and enforce it before outbound signing.
2. Forward signed end-to-end headers plus a deliberately documented safe unsigned allowlist; explicitly test `Range` if retained.
3. Require a valid signed `X-Amz-Date` for header authentication and validate scope as `<same-date>/<region>/s3/aws4_request`.
4. For explicit hexadecimal payload hashes, verify body bytes through the shared replay/body boundary and reject mismatches. Do not silently convert hashed payloads to `UNSIGNED-PAYLOAD`.
5. Preserve presigned URL support and avoid forwarding inbound auth query parameters.
6. Inject a private clock seam so skew and expiry boundaries are deterministic.

Acceptance criteria:

- Unsigned `x-amz-acl`, `x-amz-tagging`, metadata, storage-class, and conditional headers are rejected or stripped and never gain backend authorization.
- Signed equivalents survive outbound request construction.
- Missing/unsigned dates, scope-date mismatch, wrong service, and wrong terminator are rejected.
- A correctly signed request whose body is changed fails authentication; a matching hashed payload succeeds.
- Exact skew and presign-expiry boundaries are deterministic in tests.
- `go test ./internal/auth ./internal/backend/s3` passes.

Completion evidence 2026-08-09:

- Added authenticated-header filtering after successful verification, preserving signed headers and unsigned `Range` while stripping unsigned S3 control headers before backend dispatch.
- Tightened header and presigned credential scope/date validation and added a private verifier clock seam for deterministic skew/expiry tests.
- Added explicit SHA-256 payload-hash verification for header and presigned requests through the replay-body boundary.
- `go test ./internal/auth ./internal/backend/s3`: passed.

## Wave 2: Routing And Configuration Correctness

### Task ROUTE-01: Correct Multi-Route And Root Operation Semantics

Status: completed 2026-08-09

Priority: P0

Suggested agent: request-pipeline specialist

Dependencies: STREAM-01

Primary ownership:

- `internal/httpapi/handler.go`
- `internal/httpapi/handler_test.go`
- `internal/s3ops/classify.go`
- `internal/s3ops/classify_test.go`

Finding:

For `on_match = "continue"` writes, the handler retains the first response and discards later responses. If a later matched route returns `4xx` or `5xx` without a transport error, the client can receive the first route's success. Separately, every method at a bucketless root is classified as `ListBuckets`, so authenticated `POST /`, `PUT /`, and `DELETE /` return a bucket listing. `ListBuckets` also bypasses `allow_ops` because it returns before route authorization.

References:

- `internal/httpapi/handler.go:104-122`
- `internal/httpapi/handler.go:136-183`
- `internal/httpapi/handler_test.go:51-78`
- `internal/s3ops/classify.go:26-29`
- `internal/auth/auth.go:66-73`

Implementation requirements:

1. Define continued writes as all-matched-route success: any matched route HTTP or transport failure must produce failure.
2. Continue closing every non-selected response and preserve the chosen upstream error response when the contract permits it.
3. Classify only `GET /` as `ListBuckets`; unsupported root methods remain unsupported.
4. Enforce `allow_ops` for `ListBuckets` through one authorizer entry point without inventing a synthetic route.
5. Align README/design/website wording for multi-route and fan-out failure behavior in the same change.

Acceptance criteria:

- Responses `200, 403` across continued routes do not return `200`.
- A later transport/rewrite/reset failure closes the earlier retained response.
- `GET /` lists buckets; `POST /`, `PUT /`, and `DELETE /` return the controlled unsupported-operation response.
- A principal excluding `ListBuckets` receives `AccessDenied`; wildcard/default semantics remain tested.
- `go test ./internal/httpapi ./internal/s3ops ./internal/auth` passes.

Completion evidence 2026-08-09:

- Added multi-route write regressions for `200, 403` responses and later dispatch failure closing the earlier retained response.
- Classified only `GET /` as `ListBuckets`; root `POST`, `PUT`, and `DELETE` now classify as unsupported.
- Added `AllowOperation` authorization for `ListBuckets` and covered denied, wildcard, and default operation semantics.
- Aligned README, design docs, and website docs for multi-destination and multi-route write failure behavior.
- `go test ./internal/httpapi ./internal/s3ops ./internal/auth`: passed.
- `make vet test`: passed.

### Task HTTP-01: Make Addressing And Prefix Boundaries Consistent

Status: completed 2026-08-09

Priority: P1

Suggested agent: URL/path correctness specialist

Dependencies: none

Primary ownership:

- `internal/requestctx/context.go`
- `internal/requestctx/from_request_test.go`
- `internal/rewrite/apply.go`
- `internal/rewrite/apply_test.go`

Finding:

Virtual-hosted addressing compares DNS names case-sensitively and does not normalize a trailing dot, although routing's host-suffix matcher does. `strip_path_prefix` uses a raw `HasPrefix`, so `/images` strips from `/images-old/key`, unlike the strict path-prefix router contract. Request parsing also contains a duplicate unreachable path-style branch.

References:

- `internal/requestctx/context.go:37-79`
- `internal/requestctx/context.go:94-103`
- `internal/router/resolve.go:99-104`
- `internal/router/resolve.go:139-148`
- `internal/rewrite/apply.go:38-44`

Implementation requirements:

1. Normalize inbound host and configured suffix for lowercase DNS comparison and one trailing dot.
2. Preserve the extracted bucket spelling contract deliberately; document/test whether it is normalized to lowercase.
3. Apply `strip_path_prefix` only on exact path equality or a `prefix + "/"` boundary.
4. Remove the duplicated path-style fallback without changing addressing precedence.
5. Preserve escaped path bytes.

Acceptance criteria:

- Uppercase and trailing-dot virtual hosts resolve consistently.
- `/images` strips from `/images/key` but not `/images-old/key`.
- Mixed addressing mode precedence and escaped-key tests pass.
- `go test ./internal/requestctx ./internal/rewrite ./internal/router` passes.

Completion evidence 2026-08-09:

- Normalized virtual-hosted request hosts and configured suffixes to lowercase without one trailing dot before comparison; extracted virtual-hosted bucket names follow that normalized lowercase contract.
- Removed the duplicate path-style fallback while preserving virtual-hosted precedence over path-style addressing.
- Made `strip_path_prefix` observe the same exact-or-slash boundary as path-prefix routing while preserving escaped path bytes.
- `go test ./internal/requestctx ./internal/rewrite ./internal/router`: passed.
- `make vet test`: passed.

### Task CONFIG-01: Validate Authorization And Bucket References

Status: completed 2026-08-09

Priority: P1

Suggested agent: configuration-contract specialist

Dependencies: ROUTE-01

Primary ownership:

- `internal/config/validate.go`
- `internal/config/load_test.go`
- configuration reference documentation

Finding:

The design promises duplicate-label and unknown-reference validation, but duplicate bucket labels are accepted when visible names differ. Unknown `client.allow_routes`, invalid `client.allow_ops`, and unknown `visible_buckets` are accepted and silently deny or hide behavior, making policy typos hard to detect.

References:

- `docs/design.md:519-533`
- `internal/config/validate.go:8-27`
- `internal/config/validate.go:52-78`
- `internal/config/validate.go:141-224`

Implementation requirements:

1. Validate auth policy references only after routes and virtual buckets are known.
2. Reject duplicate bucket block labels and duplicate policy entries where useful diagnostics are possible.
3. Accept only supported operation names or `*` in `allow_ops`.
4. Reject unknown route and visible-bucket references while preserving `*`.
5. Keep errors contextual with auth/client/field names.

Acceptance criteria:

- One table-driven load test fails on each duplicate/typo case and passes for wildcard references.
- Existing valid examples continue to load.
- `go test ./internal/config` passes.

Completion evidence 2026-08-09:

- Added table-driven config load coverage for duplicate bucket labels, unknown `allow_routes`, invalid `allow_ops`, unknown `visible_buckets`, and duplicate auth policy entries; wildcard policy references remain accepted.
- Moved auth policy reference validation after route and virtual-bucket validation so route and visible-bucket names are known before checking clients.
- `go test ./internal/config`: passed.
- `make vet test`: passed.

## Wave 3: Resource Ownership And Exhaustion

### Task RESOURCE-01: Close And Reuse Every Upstream Response Correctly

Status: completed 2026-08-09

Priority: P1

Suggested agent: Go transport lifecycle specialist

Dependencies: STREAM-01, ROUTE-01

Primary ownership:

- `internal/dispatch/dispatch.go`
- `internal/dispatch/dispatch_test.go`
- response cleanup paths in `internal/httpapi/handler.go`

Finding:

Intermediate ordered-failover, fan-out, and multi-route responses are often closed without a bounded drain, preventing HTTP/1 connection reuse. Earlier retained multi-route bodies leak on later errors. Ordered failover can retain a superseded `5xx` body until another request completes, and response ownership is implicit across dispatch and handler.

References:

- `internal/dispatch/dispatch.go:73-110`
- `internal/dispatch/dispatch.go:121-150`
- `internal/httpapi/handler.go:136-179`

Implementation requirements:

1. Establish that dispatch owns intermediate responses and the handler owns only the returned primary.
2. Bounded-drain then close every discarded body; do not permit unbounded malicious error bodies to delay failover.
3. Drain/close a failed failover response before starting the next destination.
4. Close a retained multi-route primary on every later reset, rewrite, or dispatch failure.
5. Preserve the upstream primary failure body contract stated in `AGENTS.md`.

Acceptance criteria:

- Close-tracking tests cover `5xx -> 200`, partial fan-out, extra route responses, and every later error branch.
- Connection-state or `httptrace` tests demonstrate reuse for repeated small discarded responses.
- Oversized error-body draining is bounded.
- `go test ./internal/dispatch ./internal/httpapi` and `make test-race` pass.

Completion evidence 2026-08-09:

- Added shared bounded drain-and-close response cleanup and used it for discarded dispatch and handler responses.
- Added close/drain regressions for fan-out partial failure, extra fan-out responses, ordered-failover `5xx -> 200`, bounded oversized discarded bodies, and retained multi-route primary cleanup on reset, rewrite, and dispatch failures.
- Added a connection-state regression demonstrating repeated small discarded upstream responses reuse the initial HTTP connections.
- `go test ./internal/dispatch ./internal/httpapi`: passed.
- `make test-race`: passed.
- `make vet test`: passed.

### Task RESOURCE-02: Bound Aggregate Replay Memory And Clarify Body Ownership

Status: completed 2026-08-09

Priority: P1

Suggested agent: resource-control specialist

Dependencies: AUTH-01

Primary ownership:

- `internal/replaybody/replaybody.go`
- new `internal/replaybody/replaybody_test.go`
- composition wiring/config only as needed for an aggregate budget

Finding:

Replay buffering is bounded per request but not per process. With the default 32 MiB limit, 100 concurrent replaying uploads can retain roughly 3.2 GiB before allocation overhead. Successful replay also replaces the original body without closing it, and the package has no direct tests despite owning `Body`, `GetBody`, and `ContentLength` mutation.

References:

- `internal/replaybody/replaybody.go:11-40`
- `internal/replaybody/replaybody.go:44-70`
- `internal/httpapi/handler.go:124-134`
- `internal/dispatch/dispatch.go:69-71`
- `internal/backend/s3/client.go:107-123`

Implementation requirements:

1. Introduce a process/app-scoped aggregate replay budget, preferably charged by buffered bytes rather than only request count.
2. Define whether budget exhaustion backpressures or returns `429`/`503`; document the selected operational contract.
3. Release reservations on success, close, cancellation, read failure, and oversize rejection.
4. Close the consumed original body when ownership transfers to the installed replay body.
5. Add focused package tests for nil/no-body, known oversize, unknown-length overflow, read failure, idempotence, reset, custom `GetBody` error, default limit, and close ownership.

Acceptance criteria:

- Concurrent near-limit requests cannot exceed the configured aggregate budget within a small measured allocation tolerance.
- Reservations are not leaked on any tested exit path.
- Replay behavior remains reusable by auth, dispatch, handler, and backend without duplicate buffering.
- `go test ./internal/replaybody ./internal/auth ./internal/backend/s3 ./internal/dispatch ./internal/httpapi` and `make test-race` pass.

Completion evidence 2026-08-09:

- Added an app-scoped replay budget shared by auth payload-hash verification, handler multi-route replay, dispatcher fan-out replay, and backend unknown-length upload replay.
- Added `listener.replay_body_aggregate_max_bytes`; `0` defaults to `256 MiB`. Aggregate exhaustion returns `503 SlowDown` immediately, while per-request overflow remains `413 EntityTooLarge`.
- Added replaybody package coverage for nil/no-body, known oversize, unknown-length overflow, read failure, idempotence, reset, custom `GetBody` error, default limit, close ownership, release paths, and concurrent aggregate reservations.
- Updated README, website docs, and sandbox config for the aggregate replay budget contract.
- `go test ./internal/replaybody ./internal/auth ./internal/backend/s3 ./internal/dispatch ./internal/httpapi ./internal/config ./internal/app`: passed.
- `make test-race`: passed.
- `make vet test`: passed.

## Wave 4: Encapsulation, Lifecycle, And Operations

### Task APP-01: Make App Construction And Shutdown Side-Effect Free

Status: completed 2026-08-09

Priority: P1

Suggested agent: Go application-lifecycle specialist

Dependencies: STREAM-01, RESOURCE-01

Primary ownership:

- `internal/app/app.go`
- new `internal/app/app_test.go`
- `cmd/s3proxy/main.go`
- new CLI tests

Finding:

`App` exports mutable `Config` and `Server`, including secret-bearing runtime config. `Build` mutates the process-global slog default, while `Run` later reads whichever default is current. `Run` does not report `http.ErrServerClosed`, so external close can leave it blocked. The `validate` command performs full runtime construction and logging side effects. The transport is not retained for idle-connection cleanup.

References:

- `internal/app/app.go:38-45`
- `internal/app/app.go:62-117`
- `internal/app/app.go:120-143`
- `cmd/s3proxy/main.go:31-74`

Implementation requirements:

1. Make app fields private and keep secret-bearing config out of the exposed lifecycle API.
2. Inject/store the logger; do not call `slog.SetDefault` from reusable construction.
3. Always observe server termination and normalize expected `http.ErrServerClosed` behavior.
4. Retain the transport/client and close idle connections during shutdown.
5. Make `validate` call side-effect-free configuration loading rather than full app wiring.
6. Set `cobra.NoArgs` on commands and make command output injectable for tests.

Acceptance criteria:

- Bind failure, context cancellation, server close, shutdown timeout/error, and repeated build/shutdown paths are directly tested.
- Building or validating two apps cannot redirect each other's logs.
- Validation does not construct runtime HTTP dependencies.
- Stray positional CLI arguments are rejected.
- `go test ./internal/app ./cmd/s3proxy` passes.

Completion evidence 2026-08-09:

- Made `App` lifecycle fields private, injected/stored an app-scoped logger, and removed reusable construction's `slog.SetDefault` mutation.
- Added `app.ValidateConfig` so the CLI `validate` command loads config without constructing runtime HTTP dependencies.
- Made `Run` observe all server exits, normalize `http.ErrServerClosed`, use a configurable shutdown timeout for tests, and close retained transport idle connections on shutdown.
- Added Cobra `NoArgs` enforcement and output injection for `validate` and `version`.
- Added lifecycle and CLI regression tests for bind failure, context cancellation, external server close, shutdown timeout, repeated build/shutdown, logger isolation, injected output, and positional arg rejection.
- `go test ./internal/app ./cmd/s3proxy`: passed.

### Task ARCH-01: Narrow Runtime Boundaries And Snapshot Configuration

Status: completed 2026-08-09

Priority: P2

Suggested agent: Go package-design specialist

Dependencies: AUTH-01, ROUTE-01, APP-01

Primary ownership:

- consumer interfaces in `internal/httpapi`
- `internal/router/resolve.go`
- `internal/listbuckets/service.go`
- narrow adapters at config/backend/rewrite boundaries

Finding:

Interfaces are mostly producer-owned and broader than consumers need. The unused `AllowBucketVisibility` policy is duplicated by `listbuckets`. Router and list services retain mutable config slices/maps, and execution models carry full config routes/targets, including credentials, across package boundaries. Operation semantics are duplicated as config strings and `s3ops.Operation` constants.

References:

- `internal/httpapi/handler.go:23-43`
- `internal/auth/auth.go:55-81`
- `internal/listbuckets/service.go:16-58`
- `internal/router/resolve.go:14-36`
- `internal/backend/s3/client.go:17-23`
- `internal/rewrite/apply.go:17-19`
- `internal/config/types.go:10-17`
- `internal/config/validate.go:227-289`
- `internal/s3ops/classify.go:11-105`

Implementation requirements:

1. Define narrow consumer-owned handler interfaces for exactly the methods used.
2. Choose one bucket-visibility policy owner; remove the duplicate implementation.
3. Snapshot retained routes, parsers, targets, and buckets at construction so post-build config mutation cannot alter serving behavior.
4. Pass `RewriteRule` rather than a full route into rewrite; introduce only similarly narrow execution inputs where they reduce credential/config coupling.
5. Centralize operation vocabulary/capabilities in a dependency-neutral leaf package only if it avoids cycles without a broad migration.
6. Keep this task behavior-preserving except for previously approved contracts.

Acceptance criteria:

- Handler test doubles implement only methods the handler consumes.
- Mutating source runtime maps/slices after component construction does not change behavior; race tests remain clean.
- Credentials do not travel through matches or rewrite inputs unless the backend execution boundary requires them.
- A table proves accepted config operations and runtime capability predicates cannot diverge.
- `make vet test` and `make test-race` pass.

Completion evidence 2026-08-09:

- Added handler-owned narrow interfaces for authentication, authorization, routing, rewriting, dispatch, and bucket listing so test doubles implement only consumed methods.
- Removed duplicate bucket-visibility authorization from `auth`; `listbuckets` remains the single visibility owner.
- Snapshotted retained router routes/parsers/targets and list-bucket views at construction, with mutation regression tests.
- Changed rewriting to accept `config.RewriteRule` instead of full routes and moved operation vocabulary/capabilities into the dependency-neutral `internal/s3op` leaf package.
- `go test ./internal/httpapi ./internal/rewrite ./internal/auth ./internal/router ./internal/listbuckets ./internal/s3ops ./internal/config`: passed.
- `make vet test`: passed.
- `make test-race`: passed.

### Task OPS-01: Add Actionable Completion And Destination Telemetry

Status: completed 2026-08-09

Priority: P2

Suggested agent: observability specialist

Dependencies: ROUTE-01, RESOURCE-01, APP-01

Primary ownership:

- `internal/httpapi/handler.go`
- telemetry-focused handler tests
- dispatch result/attempt metadata only if needed

Finding:

Successful ordinary requests have no completion record with status, bytes, or latency. `dispatch.Result.Errors` contains destination-specific failures, including degraded successful failover, but the handler logs only aggregate failures. The design requires per-destination result details. `/readyz` is identical to liveness without an explicit readiness contract.

References:

- `internal/httpapi/handler.go:46-186`
- `internal/dispatch/dispatch.go:16-19`
- `internal/dispatch/dispatch.go:85-110`
- `internal/dispatch/dispatch.go:129-150`
- `docs/design.md:282-287`

Implementation requirements:

1. Capture downstream status and bytes and emit one structured completion record with request ID and duration.
2. Log destination attempt outcome for partial fan-out and successful/failed ordered failover without secrets or object payloads.
3. Preserve request ID response behavior.
4. Do not add backend polling. Either document readiness as startup readiness or inject a narrow checker after a maintainer decision.
5. Do not add metrics solely because Prometheus is present in `go.mod`; telemetry choice should follow an explicit operational contract.

Acceptance criteria:

- Completion logs cover health, auth denial, route miss, upstream success, and upstream failure.
- Destination logs identify target, route, operation, status/error, and request ID for degraded success and failure.
- Logger tests use an isolated buffer-backed handler and do not depend on global slog state.
- `go test ./internal/httpapi ./internal/dispatch` passes.

Completion evidence 2026-08-09:

- Added a response-writer recorder so every request emits one structured completion log with request ID, method, path, status, bytes, and duration.
- Added dispatch attempt metadata and handler destination-attempt logs for fan-out/failover degraded success and failure without request payloads or credentials.
- Documented `/readyz` as startup readiness only; no backend polling was added.
- `go test ./internal/httpapi ./internal/dispatch`: passed.

## Wave 5: Measured Performance And Integration Coverage

### Task PERF-01: Measure And Remove Confirmed Hot-Path Waste

Status: completed 2026-08-09

Priority: P3

Suggested agent: Go benchmarking specialist

Dependencies: RESOURCE-01, RESOURCE-02, ARCH-01

Primary ownership:

- benchmark files beside `internal/backend/s3`, `internal/router`, and `internal/dispatch`
- minimal implementation changes justified by benchmark evidence

Finding:

The shared transport sets `MaxIdleConns` but leaves `MaxIdleConnsPerHost` at Go's default of two. Fan-out is serial, so latency approaches the sum of backend latencies. Request header filtering rebuilds `Connection` token maps per header; path escaping uses `fmt.Sprintf` per escaped byte; routing allocates capture maps for non-regex parsers. No benchmarks currently establish which optimizations matter.

References:

- `internal/app/app.go:62-75`
- `internal/dispatch/dispatch.go:73-101`
- `internal/backend/s3/client.go:81-88`
- `internal/backend/s3/client.go:215-236`
- `internal/backend/s3/client.go:259-295`
- `internal/router/resolve.go:39-81`
- `internal/router/resolve.go:94-127`

Implementation requirements:

1. Set and test a deliberate `MaxIdleConnsPerHost`; use connection-count evidence under concurrent load.
2. Add `-benchmem` baselines before changing header filtering, escaping, or route allocations.
3. Parallelize fan-out only if latency benefit is demonstrated and concurrency is bounded; preserve deterministic primary/error semantics and run under `-race`.
4. Avoid caching or pooling unless profiles show a material benefit.

Acceptance criteria:

- Benchmarks report before/after allocations and latency for each accepted optimization.
- Concurrent same-host traffic reuses a deliberate pool rather than the default two idle connections.
- Any parallel fan-out is bounded, cancellation-safe, race-free, and semantically equivalent.
- `make vet test`, `make test-race`, and relevant `go test -bench . -benchmem` commands pass.

Completion evidence 2026-08-09:

- Added benchmarks for outbound header forwarding, escaped-path canonicalization, route resolution/matching, and current serial fan-out dispatch.
- Set upstream `MaxIdleConnsPerHost` to `32` and added tests for the configured value and same-host concurrent connection reuse across repeated waves.
- Reused parsed `Connection` header tokens once per outbound request, replaced `fmt.Sprintf` percent escaping with direct hex writes, and avoided capture-map allocation for non-regex parser matches.
- Benchmark evidence on linux/amd64: header forwarding improved from `18083 ns/op`, `17600 B/op`, `346 allocs/op` to `7575 ns/op`, `6640 B/op`, `112 allocs/op`; escaped-path canonicalization improved from `561.7 ns/op`, `160 B/op`, `7 allocs/op` to `138.9 ns/op`, `144 B/op`, `2 allocs/op`; path-prefix parser matching improved from `40.83 ns/op`, `48 B/op`, `1 alloc/op` to `16.99 ns/op`, `0 B/op`, `0 allocs/op`.
- Kept fan-out serial; `BenchmarkDispatchFanoutSerial` recorded `1071 ns/op`, `1271 B/op`, `25 allocs/op`, but deterministic primary/error semantics were not changed without a separate concurrency design.
- `go test ./internal/app ./internal/backend/s3 ./internal/router ./internal/dispatch`: passed.
- `go test -bench . -benchmem ./internal/backend/s3 ./internal/router ./internal/dispatch`: passed.
- `make vet test`: passed.
- `make test-race`: passed.

### Task TEST-01: Close Contract And End-To-End Coverage Gaps

Status: completed 2026-08-09

Priority: P2

Suggested agent: integration-test specialist

Dependencies: AUTH-01, ROUTE-01, HTTP-01, CONFIG-01, RESOURCE-02

Primary ownership:

- `internal/integration`
- `sandbox/integration-config.hcl`
- README/design/website contract alignment

Finding:

The documented integration strategy includes invalid signatures, virtual-hosted routing, rewrite behavior, multipart rejection, and fan-out failures, but current integration coverage omits those cases. It also omits `HeadBucket`, named-capture templates, ordered failover boundaries, replay-limit `413`, and graceful process lifecycle. Documentation conflicts on fan-out error responses, virtual bucket route meaning, and metrics availability.

References:

- `docs/design.md:626-656`
- `internal/integration/integration_test.go:75-99`
- `internal/integration/integration_test.go:251-278`
- `sandbox/integration-config.hcl:5-8`
- `README.md:251-264`
- `website/docs/api-reference.md:49-52`
- `website/docs/api-reference.md:94-101`
- `website/docs/configuration.md:289`
- `website/docs/providers-and-routing.md:81-85`

Implementation requirements:

1. Add live-stack tests for tampered signatures, virtual-hosted addressing, named-capture rewrite, multipart/CopyObject rejection, and `HeadBucket`.
2. Test fan-out primary and replica HTTP/transport failures against the selected contract.
3. Test ordered failover on `5xx` and prove `404` does not fail over.
4. Test replay-limit `413` through the running server.
5. Align documentation with implemented contracts; remove claims for absent metrics or virtual-bucket routing behavior unless implemented.
6. Keep sandbox services intact as required by `AGENTS.md`.

Acceptance criteria:

- Each listed scenario fails meaningfully on the old behavior where applicable and passes after remediation.
- Unit-only environments continue to skip build-tagged integration tests.
- `make vet test` passes.
- `make sandbox-integration-up` passes when Docker and the live sandbox are available.

Completion evidence 2026-08-09:

- Added live-stack integration coverage for tampered signatures, virtual-hosted routing, named-capture key templates, multipart initiation rejection, CopyObject rejection, HeadBucket, fan-out primary HTTP failure, fan-out later HTTP failure, fan-out transport failure, ordered failover on `5xx`, no failover on `404`, and replay-limit `413`.
- Extended the sandbox integration config with virtual-hosted addressing, named-capture routes, deterministic fan-out failure routes, a low per-request replay limit, and a deterministic sandbox `500` upstream while preserving existing sandbox services.
- Aligned docs for fan-out failure responses, non-transactional fan-out writes, virtual-bucket route references, and removed claims for metrics that are not implemented.
- `go test -tags integration -run '^$' ./internal/integration`: passed.
- `make vet test`: passed.
- `make sandbox-integration-up`: passed.

## Deferred Maintainer Decisions

These decisions do not block Wave 1 confirmed fixes unless noted:

1. Unsigned payload policy: whether `UNSIGNED-PAYLOAD` should require direct TLS, an explicitly trusted TLS-terminator mode, or remain accepted as today. AUTH-01 must still verify explicit hashes and prevent unsigned-header privilege laundering.
2. Aggregate replay exhaustion response: backpressure versus immediate `429` or `503`. RESOURCE-02 needs this decision before finalizing the external error contract.
3. Insecure backend HTTP: whether non-loopback `http://` targets require an explicit `allow_insecure_http` option. Current config is a privileged boundary, so this is hardening rather than a confirmed remote vulnerability.
4. Readiness: remove `/readyz`, document startup-only readiness, or add an injected local readiness condition. Do not infer backend polling.
5. Fan-out partial failure response: `AGENTS.md` says surface the upstream Primary response body, while README says any failure returns `502`. ROUTE-01/TEST-01 must align code and all documentation with the maintainer-selected precise behavior for primary failure, replica HTTP failure, and transport failure.
6. Listener TLS: native TLS support versus documented external termination. This is deployment hardening and should be a separate task after the trust model is selected.
7. Endpoint destination policy: if configuration becomes delegated to a less-trusted control plane, create a separate SSRF task covering DNS resolution, rebinding, redirects, and CIDR allowlists.

## Final Integration Review

### Task INTEGRATION-01: Independently Audit Remediation Completion

Status: completed 2026-08-09

Priority: P0

Suggested agent: independent senior reviewer not used for primary implementation

Dependencies: STREAM-01, AUTH-01, ROUTE-01, HTTP-01, CONFIG-01, RESOURCE-01, RESOURCE-02, APP-01, ARCH-01, OPS-01, TEST-01; PERF-01 may be deferred with evidence

Primary ownership:

- review-only across the repository
- task status and completion evidence in this file

Finding:

Cross-cutting changes affect authentication, streaming ownership, multi-route semantics, and lifecycle. An independent pass is required to detect alternate-path bypasses and documentation drift.

Implementation requirements:

1. Verify every acceptance criterion against tests and runtime behavior, not implementation claims.
2. Recheck authenticated headers across header-signed and presigned paths.
3. Recheck payload integrity, replay limits, and body closure on success, failure, cancellation, and failover.
4. Confirm public documentation and implementation agree on fan-out, virtual buckets, readiness, TLS, and unsupported operations.
5. Confirm request-controlled bodies and collections have per-request and aggregate bounds.
6. Confirm no secret-bearing config or internal attempt data crosses external/log boundaries.
7. Review all deferred tasks for explicit rationale and residual risk.

Acceptance criteria:

- `make vet test` passes.
- `make test-race` passes.
- `make build` passes.
- `make sandbox-integration-up` passes when the live sandbox is available, or the environmental blocker and unverified scenarios are recorded.
- No P0/P1 task remains pending without an explicit owner, blocker, and risk statement.
- Completion evidence is appended to every completed task.

Completion evidence 2026-08-09:

- Independently audited the remediated authentication, replay/body ownership, streaming, routing, lifecycle, documentation, and logging boundaries against the task acceptance criteria and existing tests.
- Confirmed all prior P0/P1 remediation tasks and PERF-01 have completion evidence; no P0/P1 implementation task remains pending.
- Clarified public documentation for `ListObjectsV1` unsupported behavior and `/readyz` startup-only readiness, and added residual-risk wording for deferred credential-lifecycle features.
- `make vet test`: passed.
- `make test-race`: passed.
- `make build`: passed.
- `make sandbox-integration-up`: passed.

## Definition Of Done

- All P0 and P1 tasks are completed with regression tests and command evidence.
- P2 tasks are completed or deliberately deferred with rationale and residual risk.
- P3 optimizations include benchmark evidence or are deferred without speculative changes.
- Authentication covers headers, scope/time, and explicit payload hashes across header and query signing paths.
- Streaming responses remain valid for the duration of configured target timeouts, and copy failures are observable.
- Every upstream response body has explicit ownership and is closed on every path; discarded bodies use bounded drains.
- Multi-route writes cannot report success when any required route fails.
- Replay memory has both per-request and aggregate bounds.
- Runtime components do not expose mutable secret-bearing config or depend on global logger mutation.
- Config policy typos fail startup with contextual errors.
- Documentation, unit tests, integration tests, and runtime contracts agree.
- Independent final review and required repository verification pass.
