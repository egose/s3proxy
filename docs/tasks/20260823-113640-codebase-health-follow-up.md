# Codebase Health Follow-Up

Created: 2026-08-23 11:36:40 local time

Related task: `docs/tasks/20260804-133206-codebase-health-remediation.md` (completed 2026-08-09)

## Objective And Scope

Remediate security, correctness, resource-lifecycle, performance, readability, and architectural gaps found in a follow-up review of the current Go proxy. This is a new phase because the prior remediation plan is complete; do not rewrite its historical status or completion evidence.

The first wave closes two inbound-to-outbound authorization bypasses. Later waves make replay and response ownership safe, then improve lifecycle, encapsulation, observability, and measured fan-out performance.

## Working Rules And Non-Goals

- Preserve unrelated worktree changes. The worktree was clean at review time, but agents must recheck before editing and must not revert concurrent changes.
- Add a regression test that fails on the current implementation before fixing each confirmed defect.
- Treat `SEC-01` and `SEC-02` as release blockers. Do not expose `sigv4_static` publicly until both are complete.
- Preserve escaped-path behavior, outbound `UNSIGNED-PAYLOAD`, and explicit outbound `Content-Length` signing requirements in `AGENTS.md`.
- Preserve the current non-transactional fan-out contract: every destination is attempted and any failure fails the request; no rollback is implied.
- Do not add multipart support, merged listings, hot reload, database-backed auth, or backend health polling.
- Do not turn configuration-origin endpoint selection into a remote SSRF finding. Configuration remains a privileged boundary.
- Avoid broad rewrites. Enforce each contract at the smallest shared boundary and keep source comments rare, per `AGENTS.md`.
- Update task status and append completion evidence as work proceeds. Completion requires the listed verification, not only code changes.

## Baseline Verification

Review baseline on 2026-08-23:

- `make vet test`: passed for all packages.
- `git status --short`: clean.
- `make test-race`: passed for all packages.
- `make build`: passed.
- `go test -tags integration -run '^$' ./internal/integration`: passed; the build-tagged suite compiles.
- `make sandbox-integration-up`: not run; it requires the live Docker sandbox.
- Static review confirmed that current tests do not reject a tampered presigned signature or unsupported object subresource queries.

After each task, run its targeted command. After each wave, run `make vet test`. Run `make test-race` after replay, lifecycle, or fan-out changes. Run `make sandbox-integration-up` after the security and final integration waves when Docker is available.

## Priority Definitions

- P0: release-blocking authentication, authorization, or data-integrity defect.
- P1: significant correctness, resource-exhaustion, lifecycle, or operational risk.
- P2: architectural health, testability, observability, or bounded performance work.
- P3: optional optimization that requires benchmark or profile evidence.

## Execution Waves

| Wave | Tasks                            | Parallelization                                                                                        |
| ---- | -------------------------------- | ------------------------------------------------------------------------------------------------------ |
| 1    | SEC-01, SEC-02                   | Parallel after agreeing ownership of integration tests; primary source files differ                    |
| 2    | HTTP-01, AUTH-01, CONFIG-01      | Parallel; `AUTH-01` follows `SEC-01` because both edit auth tests and verification order               |
| 3    | RESOURCE-01, RESOURCE-02, APP-01 | `RESOURCE-01` and `RESOURCE-02` may run in parallel; `APP-01` must coordinate slow-body shutdown tests |
| 4    | ARCH-01, ARCH-02, OBS-01         | Sequential where `router.Match`, `dispatch.Result`, or constructors overlap                            |
| 5    | PERF-01                          | Only after replay and response ownership are race-safe                                                 |
| 6    | INTEGRATION-01                   | Independent final reviewer only                                                                        |

Recommended agent allocation:

| Agent                    | Tasks                | Primary ownership                             |
| ------------------------ | -------------------- | --------------------------------------------- |
| SigV4 security agent     | SEC-01, AUTH-01      | `internal/auth`, auth-focused handler tests   |
| S3 protocol agent        | SEC-02               | operation classification and query policy     |
| HTTP transport agent     | HTTP-01, RESOURCE-02 | upstream transport and response ownership     |
| Replay/concurrency agent | RESOURCE-01          | `internal/replaybody`                         |
| Lifecycle agent          | APP-01               | `internal/app`                                |
| Configuration agent      | CONFIG-01            | route validation and operation capabilities   |
| Architecture agent       | ARCH-01, ARCH-02     | immutable runtime models and composition APIs |
| Observability agent      | OBS-01               | dispatch results and safe logging             |
| Performance agent        | PERF-01              | bounded fan-out and benchmarks                |
| Independent reviewer     | INTEGRATION-01       | cross-repository acceptance audit             |

## Wave 1: Authentication And Authorization

### Task SEC-01: Correct Presigned SigV4 Verification

Status: completed

Priority: P0

Suggested agent: AWS SigV4 security specialist

Dependencies: none

Primary ownership:

- `internal/auth/verify.go`
- `internal/auth/auth_test.go`
- presigned-auth cases in `internal/integration`

Finding:

`verifyQuery` calls `PresignHTTP` but discards the returned signed URI, then reads `X-Amz-Signature` from the unchanged clone. AWS SDK v2 returns a signed URI rather than mutating the request, so `generated` is the attacker-supplied signature and the comparison succeeds for any non-empty signature after structural checks. Query auth also does not require exactly one value for every signing parameter or explicitly validate `X-Amz-Algorithm`.

References:

- `internal/auth/verify.go:125-183`
- `internal/auth/auth_test.go:404-568`
- `internal/auth/auth_test.go:570-595`
- `internal/integration/integration_test.go:273-297`

Implementation requirements:

1. Capture the URI returned by `PresignHTTP`, parse its generated signature, and compare it in constant time with the separately retained supplied signature.
2. Remove the supplied signature from the signing input if required by the SDK contract; do not compare values read from the same unchanged query.
3. Require `X-Amz-Algorithm=AWS4-HMAC-SHA256` and exactly one value for every required presign authentication parameter. Reject duplicate security-sensitive parameters rather than relying on `url.Values.Get`.
4. Preserve valid long-lived presigned URLs within `X-Amz-Expires`, signed-header filtering, explicit payload hashes, and current scope/date validation.
5. Add live-stack tampering coverage; changing path, query, signed header, signature, algorithm, or duplicating a signing parameter must fail before routing.

Acceptance criteria:

- A random, one-character-changed, empty, or duplicated `X-Amz-Signature` is rejected with `SignatureDoesNotMatch` or the documented controlled auth error.
- A known access-key ID without its secret is insufficient to authenticate a presigned request.
- An unchanged SDK-generated URL remains valid at the exact expiry boundary and invalid after it.
- Unit tests prove the generated and supplied signatures originate from separate values.
- `go test ./internal/auth ./internal/httpapi` passes.
- `make sandbox-integration-up` passes with tampered-presign cases when Docker is available.

Completion evidence:

- Changed: `internal/auth/verify.go`, `internal/auth/auth_test.go`, `internal/integration/integration_test.go`.
- Verified: `go test ./internal/auth ./internal/httpapi` passed.
- Verified: `go test -tags integration -run '^$' ./internal/integration` passed; integration tamper cases compile without requiring a live stack.
- Not run: `make sandbox-integration-up`; Docker is unavailable in this environment (`docker` command not found).

### Task SEC-02: Reject Unsupported S3 Subresource Operations

Status: completed

Priority: P0

Suggested agent: S3 protocol and authorization specialist

Dependencies: none

Primary ownership:

- `internal/s3ops/classify.go`
- `internal/s3ops/classify_test.go`
- `internal/httpapi/handler.go`
- unsupported-operation integration tests

Finding:

Classification mostly considers method and key presence. Queries such as `PUT ?acl`, `PUT ?tagging`, `GET ?tagging`, or `DELETE ?tagging` are authorized as ordinary `PutObject`, `GetObject`, or `DeleteObject`. The backend then forwards every non-auth query parameter and signs it with destination credentials, allowing a client to invoke unsupported S3 APIs under a broader allowed operation.

References:

- `internal/s3ops/classify.go:27-68`
- `internal/s3ops/classify_test.go:33-163`
- `internal/httpapi/handler.go:119-170`
- `internal/backend/s3/client.go:170-212`
- `internal/backend/s3/client.go:359-382`

Implementation requirements:

1. Define the allowed query-key combinations for each supported operation at the classification boundary.
2. Classify unsupported S3 subresources as unsupported, or model them as distinct operations requiring explicit policy support. Do not silently drop a signed query and reinterpret the request.
3. Cover ACL, tagging, retention, legal hold, torrent, versioning/version IDs, restore, select, response-header overrides, and other S3 subresources relevant to the supported methods. Document intentionally allowed ordinary parameters such as `versionId` only if the operation contract truly supports them.
4. Keep multipart and CopyObject rejection behavior intact and reject the request before route dispatch.
5. Update README/design/API docs so the supported operation surface is precise.

Acceptance criteria:

- Clients allowed only current v1 operations cannot cause unsupported S3 APIs to reach a backend.
- Tests use a counting backend to prove rejected query combinations cause zero dispatches.
- Ordinary `GetObject`, `PutObject`, `DeleteObject`, `HeadObject`, and `ListObjectsV2` query behavior remains covered.
- Unit and integration tests cover at least `?acl`, `?tagging`, `?retention`, `?legal-hold`, and multipart query variants.
- `go test ./internal/s3ops ./internal/httpapi ./internal/backend/s3` passes.

Completion evidence:

- Changed: `internal/s3ops/classify.go`, `internal/s3ops/classify_test.go`, `internal/httpapi/handler_test.go`, `README.md`, `docs/design.md`, `website/docs/api-reference.md`, `website/docs/request-examples.md`.
- Verified: `go test ./internal/s3ops ./internal/httpapi ./internal/backend/s3` passed.
- Coverage: classifier tests reject unsupported subresources and response overrides; handler tests use `countingDispatcher` to prove unsupported query, multipart, and copy requests make zero dispatch calls.

## Wave 2: Protocol And Input Hardening

### Task HTTP-01: Preserve Encoded Upstream Object Bytes

Status: completed

Priority: P0

Suggested agent: Go HTTP transport specialist

Dependencies: none

Primary ownership:

- `internal/app/app.go`
- `internal/app/app_test.go`
- response forwarding tests in `internal/httpapi`

Finding:

The shared upstream transport leaves automatic compression enabled. When the inbound request has no `Accept-Encoding`, Go can add `Accept-Encoding: gzip`, transparently decompress an upstream object, and alter encoding/length headers. S3 object bytes can then differ from the backend response and its ETag metadata, with decompression amplification as an additional resource risk.

References:

- `internal/app/app.go:124-138`
- `internal/backend/s3/client.go:95-129`
- `internal/httpapi/handler.go:283-303`

Implementation requirements:

1. Disable transparent response compression on the upstream transport.
2. Continue forwarding a client-supplied, authenticated `Accept-Encoding` according to the existing header policy, but never let the Go transport silently transform object bytes.
3. Verify response body, `Content-Encoding`, `Content-Length`, and ETag consistency through the complete forwarding path.

Acceptance criteria:

- A gzip-encoded upstream object is returned byte-for-byte unchanged when the inbound request omits `Accept-Encoding`.
- Relevant headers remain consistent with the returned bytes.
- A compression-bomb-shaped test fixture is not decompressed in the proxy.
- `go test ./internal/app ./internal/backend/s3 ./internal/httpapi` passes.

Completion evidence:

- Changed: `internal/app/app.go`, `internal/app/app_test.go`, `internal/backend/s3/client_test.go`, `internal/httpapi/handler_test.go`.
- Verified: `go test ./internal/app ./internal/backend/s3 ./internal/httpapi` passed.
- Coverage: upstream transport disables transparent gzip decoding; tests cover omitted inbound `Accept-Encoding`, client-supplied `Accept-Encoding` forwarding, encoded body/header forwarding, and a compression-bomb-shaped gzip fixture remaining compressed.

### Task AUTH-01: Authenticate Before Reading Hashed Payloads

Status: completed

Priority: P1

Suggested agent: request-body authentication specialist

Dependencies: SEC-01

Primary ownership:

- `internal/auth/verify.go`
- `internal/auth/auth_test.go`
- auth-error mapping in `internal/httpapi/handler.go`

Finding:

Header and presigned verification fully buffer and hash an explicitly hashed body before comparing the request signature. Anyone knowing an access-key ID can therefore force up to the configured replay limits of allocation and hashing with an invalid signature. Replay-limit failures produced during authentication are also mapped to `403`, whereas the same errors later in the pipeline produce `413 EntityTooLarge` or `503 SlowDown`.

References:

- `internal/auth/verify.go:53-113`
- `internal/auth/verify.go:125-183`
- `internal/auth/verify.go:374-419`
- `internal/httpapi/handler.go:136-145`
- `internal/httpapi/handler.go:173-185`

Implementation requirements:

1. Verify the SigV4 signature against the claimed payload hash before reading body bytes.
2. After signature success, verify that an explicit hexadecimal hash matches the actual body and preserve replayability for dispatch.
3. Make handler mapping of replay errors consistent regardless of whether auth, multi-route handling, dispatch, or backend preparation first requested replay.
4. Preserve `UNSIGNED-PAYLOAD` behavior and do not read its body during authentication.
5. Release reservations and close owned readers on every auth failure.

Acceptance criteria:

- An invalid signature with a large or erroring body performs no body read and consumes no replay budget.
- A valid signature with altered payload bytes is rejected.
- A valid request exceeding the per-request replay limit returns `413`; aggregate exhaustion returns `503`.
- Invalid credentials and signature mismatches retain controlled authentication responses.
- `go test ./internal/auth ./internal/httpapi ./internal/replaybody` passes.

Completion evidence:

- Changed: `internal/auth/verify.go`, `internal/auth/auth_test.go`, `internal/httpapi/handler.go`, `internal/httpapi/handler_test.go`.
- Verified: `go test ./internal/auth ./internal/httpapi ./internal/replaybody` passed.
- Coverage: explicit payload hashes are signed as claims before body reads; invalid header and presigned signatures with erroring bodies read zero body bytes and use zero replay budget; valid signatures with altered bytes are rejected and release reservations; hash-verified bodies remain replayable; auth, multi-route, and dispatch replay errors map to `413`/`503` consistently while signature mismatch mapping remains preserved.

### Task CONFIG-01: Reject Ambiguous Routes And Centralize Capabilities

Status: completed

Priority: P1

Suggested agent: configuration-contract specialist

Dependencies: SEC-02

Primary ownership:

- `internal/config/load.go`
- `internal/config/validate.go`
- `internal/config/load_test.go`
- `internal/s3op/ops.go`

Finding:

Duplicate destination references survive loading and are executed independently, so `dispatch = "all"` can send the same write twice while `Result.Errors` collapses failures by target name. Route validation also duplicates read/write/fan-out operation lists instead of using the designated `internal/s3op` capability package, allowing validation and runtime dispatch to drift.

References:

- `internal/config/load.go:310-329`
- `internal/config/validate.go:149-207`
- `internal/config/validate.go:308-359`
- `internal/dispatch/dispatch.go:89-128`
- `internal/s3op/ops.go:18-51`

Implementation requirements:

1. Normalize destination references before duplicate detection and reject duplicates with route and target names.
2. Reject duplicate route operation entries for the same reason; do not silently normalize malformed policy.
3. Derive route validation from `internal/s3op` capabilities rather than private string switches.
4. Keep `internal/s3op` dependency-neutral and avoid introducing a config/classifier import cycle.

Acceptance criteria:

- Both plain and qualified duplicate target references fail startup before any backend call.
- Duplicate operations fail with contextual errors.
- A table over every declared operation proves config validation agrees with read, write, fan-out, and configurable capabilities.
- `go test ./internal/config ./internal/s3op ./internal/s3ops ./internal/dispatch` passes.

Completion evidence:

- Changed: `internal/config/validate.go`, `internal/config/load_test.go`, `internal/s3op/ops.go`, `internal/s3op/ops_test.go`.
- Verified: `go test ./internal/config ./internal/s3op ./internal/s3ops ./internal/dispatch` passed.
- Coverage: config rejects plain and qualified duplicate route destinations after normalization, rejects duplicate route operations, and validates every declared `internal/s3op` operation against read/write/fan-out/configurable capabilities.

## Wave 3: Resource And Lifecycle Safety

### Task RESOURCE-01: Make Replay Storage Race-Free And Allocation-Bounded

Status: completed

Priority: P1

Suggested agent: Go concurrency and memory specialist

Dependencies: AUTH-01

Primary ownership:

- `internal/replaybody/replaybody.go`
- `internal/replaybody/replaybody_test.go`
- replaybody benchmarks

Finding:

Context cancellation can call `payload.release`, set `payload.body = nil`, and race with unsynchronized `GetBody` reads. Buffering does not explicitly observe request cancellation while reading. The aggregate budget charges logical length rather than allocated capacity, while repeated `append` growth can retain and transiently allocate materially more memory than the advertised aggregate cap.

References:

- `internal/replaybody/replaybody.go:62-82`
- `internal/replaybody/replaybody.go:116-163`
- `internal/replaybody/replaybody.go:177-208`
- `internal/replaybody/replaybody_test.go:90-148`
- `internal/replaybody/replaybody_test.go:175-233`

Implementation requirements:

1. Make payload bytes immutable for all acquired readers, or synchronize acquisition and clearing with one ownership mechanism. Never race `GetBody` with release.
2. Make buffering context-aware, release partial reservations on cancellation, and define original-body closure in one place.
3. For known lengths, reserve atomically and allocate exactly once. For unknown lengths, charge retained capacity or use fixed chunks so accounting reflects actual retained allocation within a documented tolerance.
4. Keep release idempotent and prove readers acquired before release remain valid.
5. Add benchmarks with `-benchmem` for known and unknown 1 KiB, 1 MiB, and configured-maximum bodies.

Acceptance criteria:

- Repeated concurrent `GetBody`, context cancellation, body close, and `Release` pass under `go test -race`.
- Cancellation during a slow upload returns promptly, closes the original body, and restores the partial reservation to zero.
- Known-length buffering performs one payload allocation rather than geometric full-body growth.
- Concurrent maximum-size requests stay within a documented allocation bound above the configured aggregate budget.
- `go test ./internal/replaybody ./internal/auth ./internal/dispatch ./internal/httpapi` and `make test-race` pass.

Completion evidence:

- Changed: `internal/replaybody/replaybody.go`, `internal/replaybody/replaybody_test.go`, `internal/replaybody/replaybody_benchmark_test.go`.
- Verified: `go test ./internal/replaybody ./internal/auth ./internal/dispatch ./internal/httpapi` passed.
- Verified: `go test -run '^$' -bench 'BenchmarkEnsureReplay/(Known|Unknown)/(1KiB|1MiB|Max)$' -benchtime=1x -benchmem ./internal/replaybody` passed.
- Verified: `make test-race` passed.
- Allocation bound: known-length bodies reserve atomically and retain one exact payload slice; unknown-length bodies retain exact copied chunks charged to the aggregate budget, with one transient `32 KiB` scratch buffer per active unknown-length buffering operation outside retained-budget accounting.
- Coverage: replay readers acquired before release remain valid; concurrent saved `GetBody`, context cancellation, body close, and `Release` pass under `-race`; slow upload cancellation closes the original body and returns used budget to zero; known-length replay stores an exact-capacity contiguous payload.

### Task RESOURCE-02: Bound Response Retention And Cleanup Time

Status: completed

Priority: P1

Suggested agent: HTTP streaming lifecycle specialist

Dependencies: HTTP-01

Primary ownership:

- `internal/backend/s3/response.go`
- `internal/backend/s3/client.go`
- `internal/dispatch/dispatch.go`
- focused handler tests

Finding:

Discarding a response reads up to 512 KiB synchronously but has no independent time bound, so a backend can send headers and drip forever when target timeout is zero. In serial fan-out and continued routes, the first response remains unread while later calls run, but its target timeout remains active until body close/EOF; a successful primary can therefore expire and truncate before forwarding.

References:

- `internal/backend/s3/response.go:8-22`
- `internal/backend/s3/client.go:55-64`
- `internal/backend/s3/client.go:116-148`
- `internal/dispatch/dispatch.go:89-128`
- `internal/httpapi/handler.go:189-255`

Implementation requirements:

1. Define a byte-, time-, and cancellation-bounded discard policy. Preserve connection reuse for small finite bodies without waiting indefinitely.
2. Define primary response ownership so it cannot expire while queued behind later mandatory attempts. Options include bounded buffering for small write responses or separating upstream transaction and downstream delivery deadlines.
3. Do not remove all response-body deadlines or buffer arbitrary GET bodies.
4. Preserve primary upstream error XML when required by the fan-out contract.
5. Surface or log cleanup failures without replacing the more relevant request result.

Acceptance criteria:

- A body that never produces another byte cannot block cleanup beyond the documented bound.
- Parent cancellation ends cleanup promptly, and small finite responses still permit connection reuse.
- A primary that returns headers first remains complete after another destination delays near its timeout.
- Primary success and error bodies close on every later failure path.
- `go test ./internal/backend/s3 ./internal/dispatch ./internal/httpapi` and `make test-race` pass.

Completion evidence:

- Changed: `internal/backend/s3/response.go`, `internal/backend/s3/response_test.go`, `internal/dispatch/dispatch.go`, `internal/dispatch/dispatch_test.go`, `internal/httpapi/handler.go`, `internal/httpapi/handler_test.go`.
- Policy: cleanup drains at most `512 KiB` for up to `250 ms` or until parent cancellation, then closes; retained write responses buffer at most `1 MiB` for up to `2 s` or until parent cancellation; GET bodies remain streamed.
- Verified: `go test ./internal/backend/s3 ./internal/dispatch ./internal/httpapi` passed.
- Verified: `make test-race` passed.
- Coverage: never-producing cleanup is time bounded; parent cancellation ends cleanup promptly; small finite discarded responses still reuse connections; retained primary success and error bodies remain complete after a later destination delay; primary success and error bodies close on later failure paths; cleanup failures are logged/surfaced without replacing the request result.

### Task APP-01: Complete Shutdown Ownership On Every Exit Path

Status: completed

Priority: P1

Suggested agent: Go server lifecycle specialist

Dependencies: none

Primary ownership:

- `internal/app/app.go`
- `internal/app/app_test.go`

Finding:

`Run` returns directly when `Serve` exits and does not close the upstream idle pool. If graceful shutdown times out, it closes only idle upstream connections, does not force-close active inbound connections, and returns without observing the server goroutine. The existing timeout test releases its blocked handler only after `Run` returns, encoding the leak rather than detecting it.

References:

- `internal/app/app.go:149-182`
- `internal/app/app_test.go:170-237`

Implementation requirements:

1. Close idle upstream connections on every post-listen exit path.
2. If graceful shutdown fails or times out, force-close active server connections and wait for `Serve` to terminate.
3. Preserve the original shutdown error, joining a force-close error only when useful.
4. Ensure handler request contexts are canceled before `Run` returns from forced shutdown.

Acceptance criteria:

- External server close, unexpected `Serve` failure, normal cancellation, and shutdown timeout all release owned resources.
- A handler waiting on `r.Context().Done()` exits before `Run` returns after a shutdown timeout.
- Repeated start/timeout cycles do not increase listener, connection, or goroutine counts.
- `go test ./internal/app` and `go test -race ./internal/app` pass.

Completion evidence:

- Changed: `internal/app/app.go`, `internal/app/app_test.go`.
- Verified: `go test ./internal/app` passed.
- Verified: `go test -race ./internal/app` passed.
- Coverage: all post-listen `Run` exits close idle upstream connections; forced shutdown cancels handler contexts, force-closes active server connections, waits for `Serve`, and repeated timeout cycles do not grow listeners, connections, or goroutines.

## Wave 4: Encapsulation, Readability, And Observability

### Task ARCH-01: Return Immutable Authentication And Routing Snapshots

Status: completed

Priority: P2

Suggested agent: Go API and race-safety specialist

Dependencies: RESOURCE-01, CONFIG-01

Primary ownership:

- `internal/auth/auth.go`
- `internal/auth/verify.go`
- `internal/router/resolve.go`
- focused auth/router tests

Finding:

Authenticator construction copies `config.Client` values without cloning policy slices, and each returned principal exposes those shared slices. Router construction clones input, but returned `Match.Route` slices and target URL pointers still alias resolver-owned state. Callers can mutate policy or routing behavior across requests and introduce data races despite the intended immutable runtime snapshot.

References:

- `internal/auth/auth.go:13-18`
- `internal/auth/auth.go:33-44`
- `internal/auth/verify.go:114-122`
- `internal/auth/verify.go:185-193`
- `internal/router/resolve.go:14-19`
- `internal/router/resolve.go:39-81`
- `internal/router/resolve.go:200-227`

Implementation requirements:

1. Clone policy collections at authenticator construction and avoid exposing verifier-owned backing arrays in principals.
2. Ensure values returned from `Resolve` cannot mutate later results or resolver state.
3. Prefer immutable, narrower execution models over repeated deep copies where that reduces secret and pointer exposure.
4. Keep secrets out of principals and logs.

Acceptance criteria:

- Mutating source config after construction does not change auth or route behavior.
- Mutating one returned principal or match does not affect a later request.
- Concurrent authentication/resolution and mutation of prior returned values pass under `-race`.
- `go test ./internal/auth ./internal/router` and `go test -race ./internal/auth ./internal/router` pass.

Completion evidence:

- Changed: `internal/auth/auth.go`, `internal/auth/verify.go`, `internal/auth/auth_test.go`, `internal/router/resolve.go`, `internal/router/resolve_test.go`.
- Verified: `go test ./internal/auth ./internal/router` passed.
- Verified: `go test -race ./internal/auth ./internal/router` passed.
- Coverage: authenticator clones client policy slices at construction and returns fresh principal policy slices; router returns cloned route slices and target URL pointers; tests cover source config mutation, returned principal/match mutation, and concurrent auth/resolve with prior returned value mutation under `-race`.

### Task ARCH-02: Narrow Runtime Composition And Credential Ownership

Status: completed

Priority: P2

Suggested agent: Go package-design specialist

Dependencies: ARCH-01, SEC-02

Primary ownership:

- runtime models in `internal/router` and `internal/dispatch`
- target registry in `internal/backend/s3` or a dependency-neutral boundary
- constructors in replay-capable packages
- `internal/app/app.go`

Finding:

`config.S3Target` embeds credentials and is stored by the router, returned in matches, and passed through the HTTP handler even though only backend execution needs secrets. Separately, production correctly shares one replay budget, but convenience constructors silently create independent aggregate budgets or use a package-global default, so alternate valid compositions can violate the advertised app-wide bound.

References:

- `internal/config/types.go:96-110`
- `internal/router/resolve.go:14-37`
- `internal/httpapi/handler.go:47-57`
- `internal/dispatch/dispatch.go:28-49`
- `internal/auth/auth.go:29-44`
- `internal/backend/s3/client.go:35-52`
- `internal/httpapi/handler.go:63-70`
- `internal/app/app.go:61-89`

Implementation requirements:

1. Keep route matches limited to target identities and routing policy; resolve immutable secret-bearing target data at the dispatch/backend execution boundary.
2. Make the supported production composition require exactly one app-scoped replay budget shared by every replay-capable component.
3. Remove or clearly isolate convenience constructors that silently create separate aggregate budgets.
4. Remove unused constructor parameters and wrappers only after the ownership API settles; do not preserve unused compatibility APIs without an external consumer.
5. Keep behavior and target ordering unchanged.

Acceptance criteria:

- Router and handler tests construct matches without backend credentials or endpoint URLs.
- Credentials remain confined to the target registry/signing boundary.
- Exhausting the budget in one replay stage is observed by every other stage in a composition test.
- Public/internal constructors express all retained dependencies and no unused context, version, config, or limit parameter remains without tested behavior.
- `make vet test` and `make test-race` pass.

Completion evidence:

- Changed: `cmd/s3proxy/main.go`, `internal/app/app.go`, `internal/app/app_test.go`, `internal/auth/auth.go`, `internal/auth/auth_test.go`, `internal/auth/verify.go`, `internal/backend/s3/client.go`, `internal/backend/s3/client_test.go`, `internal/dispatch/dispatch.go`, `internal/dispatch/dispatch_test.go`, `internal/httpapi/handler_test.go`, `internal/replaybody/replaybody.go`, `internal/router/resolve.go`, `internal/router/resolve_benchmark_test.go`, `internal/router/resolve_test.go`.
- Verified: `go test ./internal/auth ./internal/router ./internal/dispatch ./internal/backend/s3 ./internal/httpapi ./internal/app ./cmd/s3proxy` passed.
- Verified: `make vet test` passed.
- Verified: `make test-race` passed.
- Coverage: router construction now consumes only route/parser data plus target names; router and handler match fixtures do not include backend credentials or endpoint URLs; SigV4 auth, dispatch, and backend construction require an explicit replay budget; tests prove budget exhausted by an earlier replay stage is observed by dispatch and backend execution boundaries.

### Task OBS-01: Make Attempt Results Authoritative And Logs Data-Safe

Status: completed

Priority: P2

Suggested agent: observability and privacy specialist

Dependencies: CONFIG-01; sequence after ARCH-02 if `router.Match` changes

Primary ownership:

- `internal/dispatch/dispatch.go`
- `internal/httpapi/handler.go`
- telemetry-focused tests

Finding:

`dispatch.Result` separately maintains `Errors` and `Attempts`, permitting drift. Single-target failures append an attempt but return `nil, err`, so destination metadata is discarded. Transport errors can contain full upstream URLs, and request completion/route-miss logs include raw paths, buckets, and keys, allowing query values or tenant/object identifiers to enter logs without an explicit policy.

References:

- `internal/dispatch/dispatch.go:16-26`
- `internal/dispatch/dispatch.go:69-81`
- `internal/dispatch/dispatch.go:89-128`
- `internal/httpapi/handler.go:87-95`
- `internal/httpapi/handler.go:158-161`
- `internal/httpapi/handler.go:208-224`
- `internal/httpapi/handler.go:346-365`

Implementation requirements:

1. Use one authoritative ordered attempt representation and derive failure counts/selection from it.
2. Return attempt metadata for single-target transport failures and log one destination failure with request ID, route, operation, and target.
3. Sanitize transport errors before logging so upstream URLs do not reveal query values or credentials.
4. Apply the maintainer-selected object identifier policy consistently: omit, hash, or explicitly opt in to raw path/bucket/key fields.
5. Preserve deterministic attempt order even if fan-out becomes concurrent later.

Acceptance criteria:

- Single-target transport failures produce destination telemetry without duplicate logs.
- Failure counts cannot disagree with attempt outcomes, including repeated target names rejected by config.
- Sentinel query values, credentials, and disallowed object identifiers do not occur in captured logs.
- Successful single-target requests do not gain noisy destination logs unless documented.
- `go test ./internal/dispatch ./internal/httpapi ./internal/backend/s3` passes.

Completion evidence:

- Changed: `internal/dispatch/dispatch.go`, `internal/dispatch/dispatch_test.go`, `internal/httpapi/handler.go`, `internal/httpapi/handler_test.go`, `internal/backend/s3/client.go`, `internal/backend/s3/client_test.go`.
- Verified: `go test ./internal/dispatch ./internal/httpapi ./internal/backend/s3` passed.
- Coverage: `Attempts` is the only authoritative ordered attempt record; single-target transport failures return attempt metadata and produce one destination failure log; transport URL credentials/query/path are redacted from logged errors; request completion and route-miss logs omit raw path/bucket/key; single-target success does not log destination attempts.

## Wave 5: Measured Performance

### Task PERF-01: Implement Bounded Parallel Write Fan-Out

Status: completed

Priority: P2

Suggested agent: Go concurrency and benchmarking specialist

Dependencies: RESOURCE-01, RESOURCE-02, OBS-01

Primary ownership:

- `internal/dispatch/dispatch.go`
- `internal/dispatch/dispatch_test.go`
- `internal/dispatch/dispatch_benchmark_test.go`
- fan-out configuration/docs only if a configurable bound is selected

Finding:

`dispatch = "all"` executes every destination serially, so latency and worst-case timeout approach the sum of backend latencies and a slow target delays healthy targets. Existing replay support permits independent readers, but the current benchmark uses instantaneous stubs and measures only serial allocation/control overhead.

References:

- `internal/dispatch/dispatch.go:85-131`
- `internal/dispatch/dispatch_benchmark_test.go:16-44`
- `internal/integration/integration_test.go:124-196`

Implementation requirements:

1. First add delayed-backend benchmark evidence comparing serial and candidate bounded-parallel execution.
2. Give each attempt its own replay reader and immutable request/target inputs; never mutate shared `req.Body` concurrently.
3. Bound concurrency and preserve configured destination order in result metadata and primary selection, independent of completion order.
4. Attempt every destination unless parent cancellation occurs; one destination failure must not cancel required peers.
5. Close every response on success, HTTP failure, transport failure, setup failure, and parent cancellation.
6. Keep multi-route parallelism out of this task because rewrite and primary-response semantics differ.

Acceptance criteria:

- Two destinations delayed by the same interval complete near one interval rather than their sum, within a non-flaky tolerance.
- Active attempts never exceed the selected bound.
- Primary/error behavior and attempt ordering are deterministic under reversed completion order.
- Parent cancellation terminates all active attempts and leaves no goroutine, body, or budget leak.
- Benchmarks cover zero-byte, small, and maximum replay-size bodies with backend delay and `-benchmem`.
- `go test ./internal/dispatch ./internal/httpapi`, `make test-race`, and `go test -bench . -benchmem ./internal/dispatch` pass.

Completion evidence:

- Changed: `internal/dispatch/dispatch.go`, `internal/dispatch/dispatch_test.go`, `internal/dispatch/dispatch_benchmark_test.go`.
- Policy: `dispatch = "all"` write fan-out now runs with an internal fixed concurrency bound of `4`; multi-route dispatch remains serial.
- Verified: `go test ./internal/dispatch ./internal/httpapi` passed.
- Verified: `make test-race` passed.
- Verified: `go test -bench . -benchmem ./internal/dispatch` passed; delayed serial vs bounded-parallel benchmarks cover zero-byte, `1 KiB`, and `32 MiB` replay-buffered bodies.
- Coverage: delayed two-target fan-out completes near one backend delay; active attempts never exceed the bound; result attempts and primary selection remain in configured destination order under reversed completion; parent cancellation stops active attempts and releases replay budget after request release; each parallel attempt receives its own replay reader.

## Deferred Maintainer Decisions

These decisions should be recorded before the dependent task starts; only the first item blocks a P0 task:

1. Supported query contract for `SEC-02`: decided for this phase by the S3 protocol owner as fail-closed for every unrecognized S3 subresource. Residual risk: operators needing versioned reads, tagging, ACLs, or multipart must wait for first-class operations rather than relying on pass-through behavior.
2. Object identifier logging for `OBS-01`: decided for this phase by the observability owner as omit raw path/bucket/key by default while retaining route/operation/request ID. Residual risk: incident debugging has less direct object context unless a future explicit opt-in or hashing policy is added.
3. Fan-out concurrency for `PERF-01`: decided for this phase by the performance owner as a fixed internal bound of `4`. Residual risk: deployments cannot tune the bound without a code change, but attempts remain bounded and deterministic.
4. Response deadline policy for `RESOURCE-02`: decided for this phase by the HTTP transport owner as bounded write-response buffering and bounded cleanup. Residual risk: write responses larger than the retention cap fail, while large `GET` bodies remain streamed and depend on downstream/client behavior.
5. Native listener TLS versus documented trusted external termination remains deployment hardening owned by maintainers. It does not block these fixes, but `UNSIGNED-PAYLOAD` should not be exposed over an untrusted cleartext hop.
6. Total active request/upstream connection limits remain operational hardening owned by maintainers. Residual risk: process-wide request pressure is bounded by replay memory and transport/server settings, not by a dedicated request concurrency limiter; add one only with a defined backpressure contract and tests that avoid cross-target head-of-line blocking.
7. `statusRecorder` support for optional interfaces and `io.ReaderFrom` remains a P3 optimization owned by maintainers. Residual risk: response forwarding may miss optional fast paths or interface-specific behavior until benchmark evidence justifies changing the wrapper.

## Final Integration Review

### Task INTEGRATION-01: Independently Audit Follow-Up Remediation

Status: completed

Priority: P0

Suggested agent: independent senior reviewer not used for primary implementation

Dependencies: SEC-01, SEC-02, HTTP-01, AUTH-01, CONFIG-01, RESOURCE-01, RESOURCE-02, APP-01, ARCH-01, ARCH-02, OBS-01; PERF-01 may be deferred with benchmark evidence and residual risk

Primary ownership:

- review-only across the repository
- integration tests and documentation corrections discovered by the audit
- completion evidence in this file

Finding:

The changes cross authentication, operation classification, body replay, response ownership, runtime composition, and telemetry. An independent pass is required to check alternate-path bypasses and ensure concurrency work did not change external S3 behavior.

References:

- `AGENTS.md`
- `docs/design.md`
- `README.md`
- `docs/tasks/20260804-133206-codebase-health-remediation.md`
- all tasks in this document

Implementation requirements:

1. Verify every acceptance criterion against code and runtime behavior, not completion notes.
2. Recheck header-signed and presigned authentication with duplicate parameters, tampered paths/queries/headers, explicit hashes, and `UNSIGNED-PAYLOAD`.
3. Recheck unsupported S3 operations through every method/query combination and prove none reaches dispatch.
4. Recheck request and response ownership under success, partial failure, cancellation, timeout, failover, and shutdown.
5. Confirm request-controlled bodies and concurrency have both per-request and aggregate bounds.
6. Confirm public docs, config validation, runtime capabilities, and integration behavior agree.
7. Confirm secrets, auth query values, and disallowed object identifiers do not cross logs or package boundaries.
8. Review every deferred item for explicit rationale, owner, and residual risk.

Acceptance criteria:

- `make vet test` passes.
- `make test-race` passes.
- `make build` passes.
- `go test -tags integration -run '^$' ./internal/integration` compiles the integration suite.
- `make sandbox-integration-up` passes when Docker is available, or the environmental blocker and unverified scenarios are recorded.
- No P0 or P1 task remains pending without an explicit owner, blocker, and risk statement.
- Every completed task contains changed files, exact commands, and results as completion evidence.

Completion evidence:

- Changed: `docs/tasks/20260823-113640-codebase-health-follow-up.md`.
- Review scope: verified every task acceptance criterion against implementation and focused runtime tests, including SigV4 header/presign tampering, duplicate presign parameters, explicit payload hashes, `UNSIGNED-PAYLOAD`, unsupported S3 query/method combinations, dispatch suppression, replay bounds, response cleanup/retention, failover, shutdown, secret confinement, and log redaction.
- Verified: `make vet test` passed for all packages.
- Verified: `make test-race` passed for all packages.
- Verified: `make build` passed and built `dist/s3proxy` with version `79aa844-dirty`.
- Verified: `go test -tags integration -run '^$' ./internal/integration` passed; integration suite compiled with no tests run.
- Not run: `make sandbox-integration-up`; Docker is unavailable in this WSL distro (`docker` command not found). Unverified live-stack scenarios are end-to-end MinIO/SeaweedFS behavior for presigned tampering, unsupported operation rejection before backend dispatch, multi-destination fan-out success/failure, ordered failover, and object byte preservation against live S3-compatible backends.
- Audit result: no P0/P1 task remains pending; every completed task records changed files, exact verification commands, and results; deferred items now record explicit owner/risk rationale.

## Definition Of Done

- Presigned authentication compares independently generated and supplied signatures and rejects malformed or duplicate signing parameters.
- Unsupported S3 subresources cannot inherit authorization from broader object operations.
- Upstream object bytes are never transparently decompressed or otherwise transformed by the proxy transport.
- Invalid signatures are rejected before costly payload reads, while valid explicit hashes still protect payload integrity.
- Replay storage is race-free, cancellation-aware, and bounded in allocated memory rather than logical bytes alone.
- Response cleanup is bounded by bytes and time; retained primary responses cannot expire behind later attempts.
- Every `App.Run` exit releases server and transport resources it owns.
- Route/auth snapshots cannot be mutated through source config or returned values.
- Credentials are confined to the backend signing boundary and one replay budget governs the full app composition.
- Configuration rejects duplicate destinations/operations and derives operation policy from one capability source.
- Attempt telemetry has one source of truth and logs conform to an explicit sensitive-data policy.
- Parallel fan-out, if implemented, is bounded, deterministic, cancellation-safe, race-free, and benchmark-justified.
- Unit, race, build, integration compile, live integration when available, documentation, and runtime contracts agree.
