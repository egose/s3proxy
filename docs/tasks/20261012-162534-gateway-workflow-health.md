# Gateway Workflow Health And Backend Credential Compatibility

Created: 2026-10-12 16:25:34 (local)

Overall status: completed

## Objective And Product Context

Improve the current V1 gateway's security boundaries, deployment feedback, cloud
credential compatibility, and virtual bucket discovery cost. The product is a
stateless S3 API gateway: clients see virtual buckets and rewritten namespaces,
while the proxy authenticates clients independently from its backend credentials,
selects configured destinations, and optionally replicates basic writes.

The common business workflows in scope are tenant-safe single-request uploads,
operator validation before rollout, access to S3 using externally issued temporary
credentials, confidential diagnostics, and predictable bucket discovery for larger
tenant catalogs. Architecture work belongs at existing classifier, configuration,
signer, and catalog boundaries rather than adding another orchestration layer.

## Analysis Coverage, Baseline, And Deduplication

- Inspected README/design and operator/API documentation; CLI, app composition,
  config loading/validation, inbound authentication, operation classification,
  request parsing, routing/rewriting, dispatch, outbound URL/signature construction,
  virtual bucket discovery, and their relevant tests. Inspected Make/package/CI
  verification commands and previous task headings, then relevant earlier findings.
- Baseline: `go version` reports Go 1.27.1 on linux/amd64.
  `go test ./internal/s3ops ./internal/auth ./internal/config ./internal/listbuckets`
  passed before changes. At plan creation, full checks, new regressions, and catalog
  benchmarks had not run; their completed results are recorded below.
- Existing completed work covers streaming/query boundaries, SigV4 path escaping,
  replay budgets, fan-out, namespace listing, config diagnostic confidentiality,
  offline topology, and onboarding. This is a new follow-up phase, not a reopening
  of those completed outcomes. Relevant references:
  - `docs/tasks/20260823-113640-codebase-health-follow-up.md`: SEC-02 and OBS-01 covered
    ordinary operation selectors and transport/object-identifier logging; the
    duplicate-header and authentication-parser paths below were not covered.
  - `docs/tasks/20260926-163541-resource-and-operator-health.md`: CLI-02 verified topology
    confidentiality, including URLs whose components currently get discarded.
  - `docs/tasks/20260926-155056-final-review-follow-ups.md`: CONFIG-02 made configuration
    parser errors safe; AUTHLOG-01 concerns runtime request authentication.
- This is a focused source/contract review, not a claim to exhaust all defects.
  Release tooling, installer scripts, website implementation, dependency CVE
  auditing, and external cloud compatibility have not been re-audited. No live
  backend stack has been launched for this phase.

## Working Rules, Priorities, And Sequence

- Run one fresh sub-agent session per task, strictly sequentially in the order
  below. The final reviewer must be independent of the implementation agents.
  Each agent reads this file and `AGENTS.md`, implements its item, runs its required
  checks, and updates only its task status and completion evidence.
- Preserve the pre-existing modification to `.github/actions/setup-tools/action.yml`.
  Do not edit `CHANGELOG.md`, commit, or revert unrelated work. Record user-visible
  compatibility changes in the relevant product docs and the release-facing notes
  in this document instead.
- Use repo-relative paths in this document and `<repo-root>` for command working
  directories. Do not record local absolute paths or actual environment secrets.
- Source style follows `AGENTS.md`; keep regression tests with their fixes. Use
  `apply_patch` for edits and format only the Go files owned by the current task.
- P1: operation/security boundary with plausible backend side effects. P2: privacy,
  operational correctness, or useful backend compatibility. P3: measured optional
  optimization. No finding in this review is classified as a proven data breach.
- Shared hotspots are `docs/design.md`, configuration docs, CLI fixtures, and HTTP
  tests. Serialization prevents conflicts; later agents preserve earlier changes.
- Sequence: HEADER-01 → AUTHLOG-01 → ENDPOINT-01 → CRED-01 → CATALOG-01 → FINAL-07.
  Dependencies below identify semantic/review prerequisites, in addition to this
  mandatory execution order.

## Shared Verification And Definition Of Done

All commands run from `<repo-root>`. Each implementation task runs its focused
commands and `make vet test` after its non-trivial changes. FINAL-07 runs the full
race suite and host build in addition to reviewing the accumulated evidence.
Local `httptest` servers and synthetic credentials provide the required boundary
and signing verification without a cloud account or Docker. Live sandbox/cloud
checks are supplementary, not a completion prerequisite for this bounded plan;
record explicitly whether they ran.

A task is completed only when its acceptance criteria and required checks pass.
Record failed regression-first runs, changed paths, successful commands, relevant
test names/measurements, and limitations in `Completion evidence`. A required
check that cannot run leaves the item blocked with its prerequisite. Add any
necessary follow-up to the current requirements or a fully specified new task
before execution; do not hide scope growth in completion notes.

The whole document is done when all six items are completed, the independent
review has a criterion-by-criterion result, all required checks pass, and the
coordinator has confirmed the statuses, evidence, and untouched changelog.

### Task HEADER-01: Reject Ambiguous Unsupported Operation Headers

Status: completed

Kind: defect

Priority: P1 — classification must not authorize a normal upload while forwarding
an unsupported copy/multipart selector to a more privileged backend.

Suggested agent: S3 request-boundary implementer (fresh session)

Dependencies: none

Primary ownership:

- `internal/s3ops/classify.go` and focused classifier tests
- `internal/httpapi/handler.go` and operation-header boundary tests
- `internal/backend/s3/operation_boundary_test.go`
- Relevant API documentation and `docs/design.md`

Finding:

`Classify` uses `Header.Get("x-amz-copy-source") != ""`; `IsMultipart` uses the
same pattern for `x-amz-multipart-upload-id`. An empty first value masks later
nonempty values, and direct callers can supply noncanonical header-map keys.
The backend copies all allowed header values after repeating the same classifier,
so an empty-first copy selector can reach upstream as an authorized PutObject.
Backend interpretation of duplicate values varies; the confirmed defect is the
unsupported-selector boundary bypass, not a demonstrated production exfiltration.
Existing tests cover a single nonempty selector, not these alternate forms.

References:

- `internal/s3ops/classify.go`: `Classify`, `IsMultipart`
- `internal/backend/s3/client.go`: `Do`, header-copy loop, `shouldForwardHeader`
- `internal/httpapi/handler.go`: classification before authentication and multipart
  detection after authentication's header filtering
- `internal/httpapi/handler_test.go`: `TestHandler_CopyObjectReturnsNotImplemented`
- `internal/backend/s3/operation_boundary_test.go`:
  `TestClientDo_RejectsMalformedSourceBeforeIO`

Requirements:

1. Detect unsupported operation-selector header presence case-insensitively,
   including empty, whitespace-only, duplicate, and mixed-case map representations.
   A PUT carrying `X-Amz-Copy-Source` must never become a normal PutObject; a request
   carrying the multipart-upload marker must remain unsupported.
2. Place shared detection in the operation boundary. Authentication filtering must
   not erase the already-observed unsupported operation. Preserve authentication
   precedence for supplied signatures and the existing query/payload contracts.
3. Return normal `501 NotImplemented` for authenticated/auth-none unsupported
   requests before routing/dispatch; direct backend calls reject before body
   preparation, signing, or network I/O. Do not add CopyObject/multipart support.
4. Document the intentionally stricter empty/duplicate selector contract.

Acceptance criteria:

- Classifier and direct-executor regressions cover empty-first and later nonempty
  values, empty-only values, mixed-case names, and ordinary requests without markers.
- Real HTTP tests cover auth-none, valid header signatures, and presigned requests;
  rejected requests cause zero backend calls and no object mutation. Invalid
  signatures retain an authentication error. Tests do not require changing the
  existing authenticated payload-hash verification order.
- Direct executor rejection occurs with zero source reads/GetBody calls. Header
  detection works before dispatch-related replay and survives auth header filtering.
- Focused tests and shared checks pass; docs state the new external contract.

Verification: `go test ./internal/s3ops ./internal/backend/s3 ./internal/httpapi`
and shared `make vet test`.

Completion evidence:

- Changed: `internal/s3ops/classify.go`, `internal/s3ops/classify_test.go`,
  `internal/httpapi/handler.go`,
  `internal/httpapi/operation_header_boundary_test.go`,
  `internal/backend/s3/operation_boundary_test.go`, `docs/design.md`,
  `website/docs/api-reference.md`, and this task's evidence/release-facing note.
  Shared case-insensitive header-presence detection now guards HTTP and direct
  execution; the handler snapshots multipart detection before authentication.
- Regression-first: the existing focused suite passed before adding tests.
  `go test ./internal/s3ops ./internal/backend/s3 ./internal/httpapi -run 'Test(Classify_CopySourceHeaderPresence|IsMultipart_HeaderPresence|ClientDo_RejectsOperationHeadersBeforeIO|ClientDo_OrdinaryPutWithoutOperationHeaders|Handler_OperationHeadersHTTP|Handler_OperationHeadersPayloadHashPrecedenceHTTP)$' -count=1`
  then failed as expected: masked/noncanonical selectors classified as ordinary
  operations, direct execution read/replayed bodies and reached transport, and
  local HTTP uploads returned 200 and changed fixture objects. Nonempty multipart
  markers removed by valid authentication also bypassed rejection. Invalid
  signatures and mismatched explicit payload hashes retained their expected errors.
- Verified after the fix: `go test ./internal/s3ops ./internal/backend/s3 ./internal/httpapi`
  passed all three packages; `make vet test` passed vet and the full unit suite.
  The new classifier tests cover empty/whitespace/duplicate values, multiple map
  casings, nil/empty value slices, and absent/similarly named header controls.
  `TestClientDo_RejectsOperationHeadersBeforeIO` asserts zero reads, `GetBody`,
  transport calls, and replay-budget use for copy PUT and multipart PUT/GET/DELETE;
  `TestClientDo_OrdinaryPutWithoutOperationHeaders` preserves ordinary uploads.
- `TestHandler_OperationHeadersHTTP` passes real HTTP auth-none, signed-marker,
  filtered header/presign, and invalid header/presign cases across single-target,
  fan-out, and continued routes: 501 or the existing 403 signature error, zero
  reads/routing/backend calls, and unchanged fixture objects. It checks the actual
  wire values and proves unsigned markers were removed by authentication.
  `TestHandler_OperationHeadersPayloadHashPrecedenceHTTP` preserves body-hash
  verification before 501 and the existing 403 on hash mismatch. Existing ordinary
  storage, query/payload-boundary, and signing-interoperability tests also pass.
- Formatting/checks: `gofmt -d internal/s3ops/classify.go internal/s3ops/classify_test.go internal/backend/s3/operation_boundary_test.go internal/httpapi/handler.go internal/httpapi/operation_header_boundary_test.go`
  produced no diff; `git diff --check` and
  `git diff --exit-code -- CHANGELOG.md .env` passed. The pre-existing
  `.github/actions/setup-tools/action.yml` modification was preserved.
- Limitations: verification used local HTTP fixtures and synthetic credentials;
  live sandbox/cloud checks were not run. Evidence establishes the unsupported
  selector bypass and its rejection, not production data exfiltration. No new
  independent follow-up was identified.

### Task AUTHLOG-01: Keep Authentication Parser Values Out Of Diagnostics

Status: completed

Kind: defect

Priority: P2 — malformed authentication parameters can enter operational logs even
though raw request values are otherwise deliberately omitted.

Suggested agent: authentication/privacy implementer (fresh session)

Dependencies: none

Primary ownership:

- `internal/auth/verify.go` and authentication diagnostic tests
- `internal/httpapi` authentication log regression tests
- `website/docs/operations.md` and `docs/design.md`

Finding:

`parseAmzDateE` returns `time.Parse` errors containing the complete invalid input;
`checkPresignWindow` formats invalid `X-Amz-Expires` values into returned errors.
Both authentication paths are logged with the raw error by `ServeHTTP`. Existing
date tests cover missing dates, unsigned dates, scope, skew, and expiry, but not
confidentiality of malformed date/expiry data in errors or logs.

References:

- `internal/auth/verify.go`: `parseAmzDateE`, `checkDateSkew`, `checkPresignWindow`
- `internal/httpapi/handler.go`: both `auth failed` logging paths
- `internal/auth/auth_test.go`:
  `TestSigV4Static_RejectsMissingUnsignedAndInvalidScopeDates`,
  `TestSigV4Static_DeterministicSkewAndExpiryBoundaries`

Requirements:

1. Return stable, useful, value-free authentication causes for invalid dates and
   invalid expiry values; discard value-bearing parser causes rather than merely
   hiding their top-level formatting. Preserve `errors.Is` classification.
2. Preserve valid signature canonicalization, date skew, seven-day expiry bound,
   zero-expiry boundary behavior, body-verification order, and HTTP error mapping.
3. Add malformed header-date and presigned date/expiry regressions through real
   authentication and captured handler logs using synthetic sentinel values.
4. Document the runtime diagnostic policy at the existing operator/design sections.

Acceptance criteria:

- Returned errors, wrapped causes, XML responses, and captured logs omit sentinel
  date/expiry values, including quoted/Unicode and numeric-overflow inputs.
- Logs retain a useful fixed failure category and request completion metadata;
  rejected requests perform no routing/backend I/O or payload verification reads.
- Existing valid signatures and exact clock/expiry boundary tests continue to pass.
- Focused tests and shared checks pass.

Verification: `go test ./internal/auth ./internal/httpapi` and shared `make vet test`.

Completion evidence:

- Changed: `internal/auth/verify.go`, `internal/auth/diagnostics_test.go`,
  `internal/httpapi/auth_diagnostics_test.go`, `docs/design.md`,
  `website/docs/operations.md`, and this task's status/evidence/release-facing note.
  Invalid dates now return the fixed `AuthError` cause `invalid X-Amz-Date`;
  all invalid expiry branches return the existing `invalid X-Amz-Expires` cause
  directly. Value-bearing parser errors are discarded at the authentication boundary.
- Regression-first: baseline `go test ./internal/auth ./internal/httpapi` passed.
  An initial test-only syntax error and XML-message punctuation expectation were
  corrected before the confirmed regression run:
  `go test ./internal/auth ./internal/httpapi -run 'Test(SigV4Static_ParserDiagnosticsValueFree|SigV4Static_ZeroExpiryAndClockBoundaries|Handler_AuthParserDiagnosticsHTTP)$' -count=1`
  failed as expected on exposed date/expiry inputs in returned errors and captured
  HTTP logs, retained date parser causes, and missing fixed categories. XML mapping,
  no-read/no-routing assertions, and the new clock/expiry boundary cases already
  passed. Failure messages did not print sentinel values.
- `TestSigV4Static_ParserDiagnosticsValueFree` passes 34 cases covering header and
  presigned dates, presigned expiry, quoted/Unicode input, trailing date data,
  invalid calendar dates, numeric overflow/underflow, negative expiry, fractional
  expiry, and the seven-day upper bound. It checks fixed causes, `errors.Is`
  through an outer wrap, absence of `time.ParseError`/`strconv.NumError`, and
  `Error`, `%s`, `%v`, `%+v`, `%#v`, and `%q` across the unwrap chain. Both source
  bodies and optional `GetBody` callbacks remain unread with zero replay-budget use.
- `TestHandler_AuthParserDiagnosticsHTTP` passes 17 real HTTP cases using the
  existing signed-payload/probe fixtures and synthetic credentials. Wire values
  reach the real authenticator; captured JSON logs and S3 XML omit rejected
  values. Logs retain the fixed warning cause and request ID, method, 403 status,
  response byte count, and duration. Responses retain `403 AccessDenied` and the
  request ID. Rejections make zero routing/backend calls, source reads, `GetBody`
  calls, or replay-budget use, and preserve the fixture object.
- `TestSigV4Static_ZeroExpiryAndClockBoundaries` passes 14 deterministic cases for
  inclusive skew, nanosecond-over-boundary rejection, zero expiry at/before/after
  signing time, and exact seven-day expiry. Existing
  `TestSigV4Static_DeterministicSkewAndExpiryBoundaries`,
  `TestSigV4Static_RejectsMissingUnsignedAndInvalidScopeDates`, valid signature,
  payload verification, and HEADER-01 HTTP regressions also pass.
- Verified after the fix: `go test ./internal/auth ./internal/httpapi` passed both
  packages; `make vet test` passed vet and the full unit suite.
  `gofmt -d internal/auth/verify.go internal/auth/diagnostics_test.go internal/httpapi/auth_diagnostics_test.go`
  produced no diff; `git diff --check` and
  `git diff --exit-code -- CHANGELOG.md .env` passed. The pre-existing
  `.github/actions/setup-tools/action.yml` modification and HEADER-01 code/tests/docs
  were preserved. No commits or additional agent sessions were created.
- Limitations: verification used local HTTP fixtures and synthetic credentials;
  live sandbox/cloud checks were not run. No blocker or independent follow-up was
  identified for this task.

### Task ENDPOINT-01: Validate The Actual Outbound Endpoint Contract Offline

Status: completed

Kind: defect

Priority: P2 — deployment validation currently accepts URL settings that outbound
construction silently discards, and hosts/ports that cannot form the intended request.

Suggested agent: configuration/CLI contract implementer (fresh session)

Dependencies: none

Primary ownership:

- `internal/config/validate.go` and endpoint validation/confidentiality tests
- `cmd/s3proxy/routes_test.go` and focused real-CLI endpoint validation tests
- `website/docs/configuration.md`, `README.md`, `docs/design.md`

Finding:

`validateTargets` checks absolute HTTP(S) URL and nonempty `Host`, but
`buildTargetURL` carries only scheme/host/path into the outbound URL. Configured
userinfo, query, and fragment are silently lost. `https://:9000` has a nonempty
Host but no hostname; out-of-range numeric ports pass URL parsing. A virtual-hosted
target with a bracketed IPv6 literal cannot be prefixed with a bucket as implemented.
The topology confidentiality fixture currently treats a URL with discarded
components as successful, so that fixture and the public contract must change
together rather than weakening validation to preserve it.

References:

- `internal/config/validate.go`: `validateTargets`
- `internal/config/load.go`: target URL parsing in `buildRuntime`
- `internal/backend/s3/client.go`: `buildTargetURL`
- `cmd/s3proxy/routes_test.go`: `TestRoutesCLI` endpoint fixture
- `internal/config/load_test.go`: existing relative URL and unsupported-scheme tests

Requirements:

1. At shared config validation, reject userinfo, query (including a bare `?`),
   fragment, opaque URL forms, empty hostname, and numeric port overflow/range
   errors with value-free target/endpoint diagnostics. Reject IPv6-literal targets
   when virtual-hosted outbound addressing cannot represent them; explain use of
   `force_path_style = true` in the safe diagnostic/docs.
2. Preserve normal HTTP(S) DNS/IPv4 endpoints, bracketed IPv6 in path-style mode,
   optional ports, and escaped endpoint base paths. Use syntax-only checks, without
   DNS, backend health calls, or binding; do not add a broad network allowlist.
3. Update old success fixtures to valid private base paths and add negative
   literal/env fixtures for the newly rejected components. Cover `Load`, direct
   `Validate`, and real CLI `validate`, `routes`, and `serve` failure before runtime.
4. Explain the compatibility change, migration to supported endpoint forms, and
   separation of endpoint URL from credential blocks in product docs.

Acceptance criteria:

- Unsupported components and malformed endpoint host/port forms fail offline with
  no success/topology output, no startup logging, no backend calls, and no leaked
  sentinel values in any diagnostic formatting/unwrap path.
- Valid unavailable/occupied endpoints still validate offline; private base paths
  remain absent from topology. Existing escaped-path outbound tests pass.
- Tests and docs agree on URL component and IPv6 addressing rules; no DNS-name
  policy or automatic URL rewriting is introduced.
- Focused tests and shared checks pass.

Verification: `go test ./internal/config ./cmd/s3proxy ./internal/backend/s3 ./internal/app`
and shared `make vet test`.

Completion evidence:

- Changed: `internal/config/validate.go`, `internal/config/endpoint_test.go`,
  `internal/config/confidentiality_test.go`, `cmd/s3proxy/routes_test.go`,
  `cmd/s3proxy/endpoint_test.go`, `internal/backend/s3/endpoint_test.go`,
  `README.md`, `website/docs/configuration.md`, `docs/design.md`, and this task's
  status/evidence/release-facing note. Production enforcement is confined to shared
  configuration validation; endpoint and credential values are not rewritten.
- Baseline: `go test ./internal/config ./cmd/s3proxy ./internal/backend/s3 ./internal/app`
  passed all four packages before changes. The initial regression run,
  `go test ./internal/config ./cmd/s3proxy ./internal/backend/s3 -run 'Test(TargetEndpointInvalid|TargetEndpointDirectURLFields|TargetEndpointSyntaxErrorsValueFree|TargetEndpointOfflineContract|TargetEndpointCLI|RoutesCLI|EnvCompiledValuesRemainLiteral|BuildTargetURL_ValidatedEndpoints)$' -count=1`,
  failed on the confirmed validation gaps and two fixture assumptions: Go 1.27
  already rejects empty brackets and unbracketed IPv6 during URL parsing. Those
  fixtures were corrected to retain the existing safe parser categories.
- Confirmed regression-first:
  `go test ./internal/config ./cmd/s3proxy -run 'Test(TargetEndpointInvalid|TargetEndpointDirectURLFields|TargetEndpointCLI)$/(userinfo|bare_query|empty_fragment|missing_hostname|high_port|numeric_overflow|virtual_IPv6)$' -count=1`
  failed as expected: unsupported endpoints passed `Load`/direct `Validate`,
  `validate`/`routes` emitted success/topology, and `serve` reached runtime logging.
  Failure messages did not print sentinel values. Before the fix,
  `go test ./internal/config ./cmd/s3proxy ./internal/backend/s3 -run 'Test(TargetEndpointSyntaxErrorsValueFree|TargetEndpointOfflineContract|RoutesCLI|EnvCompiledValuesRemainLiteral|BuildTargetURL_ValidatedEndpoints)$' -count=1`
  passed the existing syntax, valid offline, migrated confidentiality, and outbound
  compatibility controls.
- `TestTargetEndpointInvalid` now passes 24 endpoint forms through direct
  `Validate` and literal/environment `Load`. `TestTargetEndpointDirectURLFields`
  covers 13 manually supplied parsed-URL cases, including `ForceQuery`, standalone
  `RawFragment`, opaque data with a host, and malformed host/port fields.
  `TestTargetEndpointSyntaxErrorsValueFree` covers 11 malformed URL forms in both
  literal and environment configurations. Errors identify target/endpoint/cause;
  `Error`, `%s`, `%v`, `%+v`, `%#v`, and `%q` across an outer wrap and the full
  unwrap chain omit private values, and URL/integer parser errors are not retained.
- `TestTargetEndpointCLI` passes 20 invalid forms through literal/environment
  `validate`, `routes`, and `serve`: exit 1, empty stdout, safe diagnostics, no
  startup logging, and zero observed backend connections. Its 11 valid endpoint
  forms pass both offline commands with an occupied listener, including an
  unhealthy local endpoint, unavailable DNS/IPv4, and path-style IPv6 with zones
  and IPv4-mapped addresses. `TestRoutesCLI` retains its exact DTO, ordering,
  defaults, byte-identical output, and confidentiality assertions using a valid
  escaped private base path. The environment-literal success fixture was migrated
  similarly in `TestEnvCompiledValuesRemainLiteral`.
- `TestTargetEndpointOfflineContract` passes 21 valid forms in literal/environment
  configurations, preserving optional/empty/boundary ports, leading zeros,
  single-label/underscore/trailing-dot/Unicode hostnames, and escaped paths without
  mutation. `TestBuildTargetURL_ValidatedEndpoints` passes eight config-to-builder
  controls for DNS/IPv4 addressing modes and path-style IPv6, preserving host/port
  and escaped base/key bytes. Existing `TestClient_S3EscapedKeySignatures` and
  escaped-path outbound tests also pass.
- Verified after implementation: `go test ./internal/config ./cmd/s3proxy ./internal/backend/s3 ./internal/app`
  passed all four packages; `make vet test` passed vet and the full unit suite.
  `gofmt -d internal/config/validate.go internal/config/endpoint_test.go internal/config/confidentiality_test.go cmd/s3proxy/endpoint_test.go cmd/s3proxy/routes_test.go internal/backend/s3/endpoint_test.go`
  produced no diff; `git diff --check` and
  `git diff --exit-code -- CHANGELOG.md .env` passed. The pre-existing
  `.github/actions/setup-tools/action.yml` edit and HEADER-01/AUTHLOG-01 changes
  were preserved. No commits or additional agent sessions were created.
- Limitations: verification used local fixtures and synthetic credentials; live
  sandbox/cloud checks were not run. No blocker or independent follow-up was
  identified for this task.

### Task CRED-01: Support Externally Issued Backend Session Credentials

Status: completed

Kind: improvement

Priority: P2 — AWS STS and compatible temporary backend credentials require a
session token, currently unavailable in the static credential configuration.

Suggested agent: backend credential/signing implementer (fresh session)

Dependencies: ENDPOINT-01 (shared credential/endpoint documentation and CLI fixtures)

Primary ownership:

- `internal/config/types.go`, `internal/config/load.go`, credential/env tests
- `internal/backend/s3/signer.go`, outbound signing/token-isolation tests
- Focused HTTP workflow and CLI confidentiality tests
- `website/docs/configuration.md`, `website/docs/operations.md`, `README.md`,
  `docs/design.md`

Finding:

`rawCredential` and `StaticCredential` only carry access/secret keys, and
`signRequest` constructs `aws.Credentials` without `SessionToken`. Operators cannot
use externally issued temporary backend credentials even though the AWS SDK signer
supports them. This is an additive configuration feature, separate from the
intentionally deferred credential issuance/refresh and inbound client lifecycle.

References:

- `internal/config/load.go`: `rawCredential`, credential loop in `buildRuntime`
- `internal/config/types.go`: `StaticCredential`
- `internal/backend/s3/signer.go`: `signRequest`
- `internal/backend/s3/client.go`: `shouldForwardHeader`, `filteredQuery`, target snapshots
- `internal/testutil/s3signature/signature.go`: independent outbound signature oracle

Requirements:

1. Add optional `session_token` to backend `credential "static"` blocks and carry
   its literal/env value through config snapshots to the AWS SDK signing credentials.
   Omitted/empty means long-lived key behavior; existing configs remain valid.
2. Only the selected target's configured token may appear on its outbound request.
   Inbound token headers/query values must still be stripped, must not override a
   backend token, and must not leak to a tokenless backend or another destination.
3. Verify that the configured token participates in outbound SigV4 signed headers,
   with independent signature verification and negative token-tampering evidence.
4. Keep credential values out of topology and diagnostics, including new HCL type
   errors with environment-derived tokens. Document how to configure temporary
   credentials and that expiry/replacement/restart remain operator-owned; this
   feature does not perform STS calls, ambient credential discovery, or refresh.

Acceptance criteria:

- Literal and environment tokens load unchanged; absent/empty tokens preserve the
  previous signing behavior. The optional field is backend-only.
- Captured local upstream requests prove correct signed token delivery and
  isolation from inbound tokens, including tokenless targets and multiple targets.
- An independent signature oracle accepts the emitted request with its configured
  token and rejects a token mutation. Real HTTP workflow coverage exercises config
  or app composition through the existing signer/executor boundaries.
- CLI topology/error output and runtime logs omit synthetic credential sentinels.
- Focused tests and shared checks pass; docs include the feature and lifecycle limit.

Verification: `go test ./internal/config ./internal/backend/s3 ./internal/httpapi ./cmd/s3proxy`
and shared `make vet test`.

Completion evidence:

- Changed: `internal/config/types.go`, `internal/config/load.go`,
  `internal/config/session_token_test.go`, `internal/backend/s3/signer.go`,
  `internal/backend/s3/session_token_test.go`, `internal/app/session_token_test.go`,
  `cmd/s3proxy/routes_test.go`, `cmd/s3proxy/session_token_test.go`, `README.md`,
  `website/docs/configuration.md`, `website/docs/operations.md`, `docs/design.md`,
  and this task's status/evidence/release-facing note. Production changes add the
  optional backend field and carry it into the existing SDK signer. Existing
  case-insensitive inbound token filtering required no modification; no dependencies
  were added.
- Baseline: `go test ./internal/config ./internal/backend/s3 ./internal/httpapi ./cmd/s3proxy ./internal/app`
  passed all five packages. Feature-first:
  `go test ./internal/app ./cmd/s3proxy -run 'Test(Build_BackendSessionTokenWorkflowHTTP|RoutesCLI)$' -count=1`
  failed because session-token configuration was unsupported. After adding the
  schema, config/CLI token tests passed. Fixture assumptions about auth-none client
  blocks, single-destination logging, and automatic `Accept-Encoding` were corrected.
  The confirmed signer regression,
  `go test ./internal/backend/s3 ./internal/app -run 'Test(ClientDo_BackendSessionTokenIsolationAndSnapshot|Build_BackendSessionTokenWorkflowHTTP|Build_BackendSessionTokenTransportDiagnosticsHTTP)$/(absent|none|header)$' -count=1`,
  then failed on missing configured tokens/signed headers and bypassed token-header
  transport validation; tokenless controls passed. Failure messages withheld values.
- `TestLoad_BackendSessionTokenValues` passes 11 cases through `Load` and `LoadFile`:
  literal/env values, HCL-looking text, whitespace, quotes/Unicode, omitted tokens,
  and empty/unset values. `TestLoad_BackendSessionTokenSnapshot` verifies environment
  changes leave loaded values intact, targets sharing credentials remain independent,
  and a fresh load sees replacements. `TestLoad_SessionTokenIsBackendOnly` rejects
  the field on clients.
  `TestLoad_BackendSessionTokenDiagnostics` passes 12 literal/env type, object-key,
  duplicate-key, index, and function-argument cases. Source/category and environment
  block/field context survive; `Error`, `%s`, `%v`, `%+v`, `%#v`, and `%q` across
  an outer wrap and the unwrap chain omit credential sentinels.
- `TestClientDo_BackendSessionTokenIsolationAndSnapshot` checks 60 captured local
  GET/PUT requests across four credential targets and six inbound header/query
  representations, including noncanonical casing and empty-first duplicates.
  The independent `internal/testutil/s3signature` oracle verifies explicit expected
  signed headers and rejects token mutation/removal. Omitted/empty targets receive
  no token; alternating targets, caller registry/URL/credential mutation, and source
  request reuse preserve isolation. Signed upload length, body bytes, unsigned
  payloads, escaped S3 paths, and zero retained replay budget are checked.
- `TestBuild_BackendSessionTokenWorkflowHTTP` passes nine auth-none/header/presign
  and temporary/tokenless/fan-out combinations through real config loading and app
  composition. Local HTTP captures verify literal/env target tokens and independent
  signatures after the environment changes, including three-destination uploads,
  single-backend reads, upstream failures, and rejected invalid inbound signatures.
  `TestBuild_BackendSessionTokenTransportDiagnosticsHTTP` verifies a non-HTTP-safe
  configured token produces 502 before backend calls without exposing values.
  Responses and captured startup/request/auth/dispatch logs omit credentials and
  inbound tokens while retaining useful completion/failure metadata.
- `TestBackendSessionTokenCLI` verifies literal/env/omitted/empty/unset success for
  real `validate`/`routes` and safe token-expression failures for `validate`, `routes`,
  and `serve`, with zero backend connections. `TestRoutesCLI` retains exact schema,
  ordering/defaults, byte-identical output, prior confidentiality assertions, and
  ENDPOINT-01's valid escaped private-path fixture while adding token coverage.
- Verified: `go test ./internal/config ./internal/backend/s3 ./internal/httpapi ./cmd/s3proxy`
  passed all four packages; `go test ./internal/app` passed; `make vet test` passed
  vet and the full unit suite. Targeted race verification,
  `go test -race ./internal/backend/s3 ./internal/app -run 'Test(ClientDo_BackendSessionTokenIsolationAndSnapshot|Build_BackendSessionTokenWorkflowHTTP|Build_BackendSessionTokenTransportDiagnosticsHTTP)$' -count=1`,
  passed both packages. Existing endpoint, confidentiality, HEADER-01, AUTHLOG-01,
  and S3 signing interoperability regressions pass in the required suites.
- Formatting/checks: `gofmt -d internal/config/types.go internal/config/load.go internal/config/session_token_test.go internal/backend/s3/signer.go internal/backend/s3/session_token_test.go internal/app/session_token_test.go cmd/s3proxy/routes_test.go cmd/s3proxy/session_token_test.go`
  produced no diff; `git diff --check` and
  `git diff --exit-code -- CHANGELOG.md .env` passed. The pre-existing
  `.github/actions/setup-tools/action.yml` edit and preceding task changes were
  preserved. No commits or additional agents were created.
- Limitations: verification used local HTTP fixtures and synthetic credentials;
  live sandbox/cloud checks were not run. External credential validity/expiry is
  not established by these tests. Docs describe the static startup snapshot, no
  automatic refresh, and complete credential replacement/restart before expiry.
  No blocker or independent follow-up was identified.

### Task CATALOG-01: Remove Repeated Sorting And Nested Visibility Scans

Status: completed

Kind: improvement

Priority: P3 — static virtual bucket catalogs should not repeat configuration-owned
ordering work or quadratic visibility checks on every discovery request.

Suggested agent: Go catalog/performance implementer (fresh session)

Dependencies: none

Primary ownership:

- `internal/listbuckets/service.go`, behavior tests, focused benchmarks
- `docs/design.md` catalog implementation rationale

Finding:

`New` snapshots a static catalog, but `List` scans `VisibleBuckets` for every
bucket and sorts the result on each call. Dense explicit policies cost O(B×V)
membership checks plus repeated sorting. Existing tests prove basic visibility,
order, and input snapshots, but do not measure scaling or ensure returned slices
cannot corrupt an optimized shared catalog. This is an optional, measured
optimization; no latency claim has yet been established.

References:

- `internal/listbuckets/service.go`: `New`, `List`, `isVisible`
- `internal/listbuckets/service_test.go`: `TestList_Sorted`,
  `TestListSnapshotsBucketsAtConstruction`
- `internal/httpapi/handler.go`: `handleListBuckets`

Requirements:

1. Add relevant benchmarks and record before/after `-benchmem` measurements for
   small and large catalogs, nil/wildcard principals, sparse explicit visibility,
   and dense explicit visibility. Keep measurements outside pass/fail timing assertions.
2. Move catalog ordering to construction and eliminate nested full policy scans
   with a bounded lookup strategy. Keep ownership simple and encapsulated; do not
   add global/per-credential caches or change authorization semantics.
3. Preserve lexicographic results, startup creation dates, nil/wildcard/empty policy
   behavior, and independent caller-owned returned slices. Visibility continues to
   be independent of route permissions as documented.
4. Verify concurrent readers and caller mutation cannot alter later results.

Acceptance criteria:

- The same focused benchmark before/after demonstrates improved large-catalog
  scaling with concrete time/allocation results, and small-catalog tradeoffs are
  recorded honestly. If no benefit is measured, revise the implementation rather
  than claiming an unmeasured speedup.
- Regression tests cover visibility/order, input/result mutation isolation, and
  concurrent discovery; race checks pass.
- No new cross-package mutable catalog or principal cache is exposed.
- Focused tests and shared checks pass.

Verification: `go test ./internal/listbuckets ./internal/httpapi`,
`go test -race ./internal/listbuckets`,
`go test ./internal/listbuckets -run '^$' -bench BenchmarkList -benchmem -count=3`
before/after, and shared `make vet test`.

Completion evidence:

- Changed: `internal/listbuckets/service.go`,
  `internal/listbuckets/service_test.go`,
  `internal/listbuckets/service_benchmark_test.go`, `docs/design.md`, and this
  task's status/evidence. Construction sorts the private bucket-view snapshot;
  explicit discovery builds a call-local visibility set and filters in catalog
  order. Nil/wildcard discovery copies the catalog, and every nonempty result has
  independent storage. Filtered allocation starts at the first match and is sized
  to the smaller of catalog size and distinct policy names. Principal state is
  not retained; route authorization remains independent of discovery visibility.
- Baseline: `go test ./internal/listbuckets ./internal/httpapi` passed before
  changes. An initial missing closing brace in the new benchmark fixture was
  corrected before measurement; the same focused command then passed the expanded
  behavior tests against the original implementation. This optimization preserves
  existing semantics, so no behavior regression was expected on that baseline.
- `TestList_VisibilityAndStartupDate` covers 11 nil/wildcard/explicit/empty policy
  cases, duplicates, unknown names, exact visible-name matching, lexicographic
  order, exact startup dates (including nanoseconds and location), and route-grant
  independence. `TestList_EmptyCatalog` covers nil and empty catalogs under five
  principal forms. `TestListSnapshotsBucketsAtConstruction` verifies source order
  and mutation isolation. `TestList_PrincipalOwnership` verifies unchanged policy
  inputs, fresh visibility after in-place policy mutation, different policies for
  equal principal identities, and preservation of an earlier result.
- `TestList_ResultMutationIsolation` mutates names/dates and appends within a
  returned slice for nil, wildcard, subset, and dense policies; another retained
  result and later filtered/full discovery remain intact.
  `TestList_ConcurrentDiscoveryWithResultMutation` runs eight synchronized readers,
  each performing 100 rounds of four shared-principal policies and mutating its
  own results. The focused race check passes.
- Verified after implementation: `go test ./internal/listbuckets ./internal/httpapi`,
  `go test -race ./internal/listbuckets`, and `make vet test` all passed.
  `gofmt -d internal/listbuckets/service.go internal/listbuckets/service_test.go internal/listbuckets/service_benchmark_test.go`
  produced no diff; `git diff --check` and
  `git diff --exit-code -- CHANGELOG.md .env` passed. Before/after `git hash-object`
  fingerprints matched for the pre-existing `.github/actions/setup-tools/action.yml`
  edit and the preceding tasks' other code/product-doc files. Their shared design
  additions were preserved. No commits or additional agents were created.
- Measurement: the identical command
  `go test ./internal/listbuckets -run '^$' -bench BenchmarkList -benchmem -count=3`
  passed before and after the service change, with a 600-second tool timeout for
  each run. Both ran from `<repo-root>` on Go 1.27.1, linux/amd64, an Intel Core
  i9-13950HX, with the default benchmark settings (`-32` suffix), sequentially
  without concurrent verification commands. The benchmark file was unchanged
  between runs. Catalogs contain 8 or 4,096 deterministically shuffled names
  (seed 1); sparse policies select every hundredth name (1 or 41 entries), dense
  policies list all names explicitly, and wildcard is `["*"]`. Fixture and service
  construction are outside the timed loop; per-call lookup construction, filtering,
  result allocation/copy, and a result-length check are included. Raw output remains
  in the execution logs; the repo-scoped measurements are summarized below.

Each measurement is the median [minimum–maximum] of three samples. A single value
means all three samples agree, including the allocation counts.

| Catalog / policy | Before ns/op                       | After ns/op               | Before B/op               | After B/op                | Before → after allocs/op |
| ---------------- | ---------------------------------- | ------------------------- | ------------------------- | ------------------------- | ------------------------ |
| 8 / Nil          | 1,062 [1,038–1,077]                | 209.5 [206.2–209.8]       | 728                       | 320                       | 7 → 1                    |
| 8 / Wildcard     | 1,172 [1,044–1,304]                | 201.0 [195.5–205.0]       | 728                       | 320                       | 7 → 1                    |
| 8 / Sparse       | 163.9 [162.0–166.3]                | 180.4 [179.0–186.1]       | 72                        | 48                        | 2 → 1                    |
| 8 / Dense        | 1,212 [1,161–1,256]                | 498.2 [438.5–995.5]       | 728                       | 320                       | 7 → 1                    |
| 4,096 / Nil      | 1,376,794 [1,341,356–1,667,837]    | 240,477 [230,064–303,259] | 532,254 [532,252–532,255] | 163,841 [163,841–163,843] | 18 → 1                   |
| 4,096 / Wildcard | 1,430,977 [1,376,076–1,487,736]    | 218,819 [217,615–242,811] | 532,253 [532,251–532,254] | 163,841                   | 18 → 1                   |
| 4,096 / Sparse   | 665,839 [608,611–684,984]          | 110,378 [84,245–123,760]  | 5,912                     | 3,624                     | 10 → 4                   |
| 4,096 / Dense    | 26,652,545 [25,792,644–28,487,794] | 477,355 [443,313–508,819] | 532,250 [532,248–532,250] | 382,276                   | 18 → 18                  |

- Interpretation/tradeoffs: all four large-catalog cases improved in these samples.
  Dense visibility no longer performs nested full-policy scans; its median falls
  from 26.653 ms to 0.477 ms. Its allocation count remains 18 because the call-local
  set adds allocations, while bytes fall from 532,250 to 382,276. The 8-bucket
  sparse case is slower by 16.5 ns/op median (10.1%), despite reducing 72 to 48 B/op
  and two allocations to one. This is the measured small-policy cost of the simple
  per-call lookup. The ranges retain observed timing variability, including the
  8-bucket dense post-change spread; there are no timing-based pass/fail assertions.
- Limitations: these are single-host, synthetic `List` microbenchmarks with three
  samples per case, not end-to-end HTTP latency or concurrent-throughput measures.
  Startup sorting cost, authentication, XML rendering, backend work, and process
  peak memory are outside their scope. Unit verification uses local fixtures;
  live sandbox/cloud checks were not run. No blocker or independent follow-up was
  identified for this task.

### Task FINAL-07: Independently Audit Workflow Health And Completion

Status: completed

Kind: investigation

Priority: P1 — independent acceptance gate for operation boundaries and credential
separation, plus complete task/evidence review.

Suggested agent: independent Go/S3 integration reviewer (fresh session)

Dependencies: HEADER-01, AUTHLOG-01, ENDPOINT-01, CRED-01, CATALOG-01

Primary ownership:

- All changed production code/tests/docs as reviewer
- This task's acceptance matrix and final verification evidence
- Focused corrections only when needed to satisfy existing task criteria

Finding:

The combined changes touch request classification before/after authentication,
offline configuration validation, backend signing, and immutable catalog state.
Per-task tests alone do not establish that these boundaries compose or that
documented completion claims match the implemented behavior.

References: the five task findings, requirements, and completion evidence above.

Requirements:

1. Inspect each acceptance criterion against code, regressions, and documentation;
   record a concise per-task criterion matrix, not just the implementers' summaries.
2. Review alternate paths: header/presigned/auth-none HTTP, direct backend calls,
   offline CLI validation, tokenless/multi-target signing, and concurrent catalogs.
3. Confirm the intentional contract changes are documented and secrets/internal
   values do not cross response/log/topology boundaries unexpectedly.
4. Run `make vet test`, `make test-race`, and `make build`. Review `git diff --check`
   and the entire intended diff; verify `CHANGELOG.md` remains unchanged and the
   pre-existing workflow edit remains outside this work.
5. Fix bounded acceptance defects with meaningful regressions and rerun affected
   checks. Surface independent new work as fully specified follow-up tasks for
   sequential execution before signing off; keep historical findings intact.

Acceptance criteria:

- All implementation tasks satisfy their criteria, with reproducible evidence and
  no unsupported completion claims or unresolved required checks.
- Full vet/unit/race/build checks pass and diff whitespace is clean.
- Product docs, configuration, runtime behavior, and test expectations agree.
- All task statuses are completed only after verification, and no changelog edit
  or unrelated-work overwrite is present.

Verification: independent criterion review, `make vet test`, `make test-race`,
`make build`, `git diff --check`, `git diff --exit-code -- CHANGELOG.md`.

Acceptance matrix:

AC numbers refer to each task's acceptance bullets in their original order.
The review inspected production control flow and ownership as well as assertions;
the full unit/race checks below include every focused package in this plan.

| Task / criterion                                                   | Result | Independent acceptance basis                                                                                                                                                                                                                                                                                                                                                                                                        |
| ------------------------------------------------------------------ | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| HEADER-01 AC1 — selector representations and ordinary controls     | Pass   | `internal/s3ops/classify.go` checks case-insensitive map-key presence, independent of values. `internal/s3ops/classify_test.go` and `internal/backend/s3/operation_boundary_test.go` cover empty-first/later, empty-only, whitespace, mixed/multiple casings, nil/empty slices, and absent/similarly named controls.                                                                                                                |
| HEADER-01 AC2 — HTTP paths, authentication precedence, no mutation | Pass   | `internal/httpapi/handler.go` retains classification and a multipart boolean before authentication mutates headers. `TestHandler_OperationHeadersHTTP` exercises real auth-none, signed/filtered header, presigned, and invalid-signature requests with zero routing/backend calls and unchanged objects. `TestHandler_OperationHeadersPayloadHashPrecedenceHTTP` preserves explicit body-hash verification and its 403 before 501. |
| HEADER-01 AC3 — rejection before source/replay work                | Pass   | `internal/backend/s3/client.go` checks shape/classification/multipart before target lookup, body preparation, signing, or transport. `TestClientDo_RejectsOperationHeadersBeforeIO` asserts zero source reads, `GetBody`, transport, and replay-budget use; HTTP rejection precedes multi-route/fan-out replay and survives unsigned-header filtering.                                                                              |
| HEADER-01 AC4 — checks and external contract                       | Pass   | Classifier/backend/HTTP suites pass in final checks. `website/docs/api-reference.md` and `docs/design.md` explicitly require omitting selectors entirely and preserve signature, strict-query, streaming-payload, and explicit-hash precedence.                                                                                                                                                                                     |
| AUTHLOG-01 AC1 — value-free errors, causes, XML, and logs          | Pass   | `internal/auth/verify.go` returns fixed `AuthError` values and discards parser errors. `internal/auth/diagnostics_test.go` inspects formatting and unwrap chains, including quoted/Unicode and overflow input; `internal/httpapi/auth_diagnostics_test.go` inspects actual HTTP XML and JSON logs. No value-bearing parser cause remains reachable.                                                                                 |
| AUTHLOG-01 AC2 — useful metadata and no request work               | Pass   | The HTTP diagnostic assertions require the fixed warning cause, request ID, method, 403, byte count, and duration, plus zero source reads, `GetBody`, routing/backend calls, and replay use. Handler response mapping remains `AccessDenied`.                                                                                                                                                                                       |
| AUTHLOG-01 AC3 — valid boundaries                                  | Pass   | Date parsing layout and window arithmetic are unchanged. `TestSigV4Static_ZeroExpiryAndClockBoundaries`, existing deterministic skew/expiry tests in `internal/auth/auth_test.go`, and signing/payload interoperability suites preserve inclusive skew, exact seven-day/zero-expiry boundaries, and signature-before-body verification.                                                                                             |
| AUTHLOG-01 AC4 — checks                                            | Pass   | Auth/HTTP and full verification pass; the fixed-category diagnostic policy in `website/docs/operations.md` and `docs/design.md` agrees with the error/log paths.                                                                                                                                                                                                                                                                    |
| ENDPOINT-01 AC1 — safe offline failures                            | Pass   | `internal/config/validate.go` rejects unsupported parsed components, the otherwise-lost bare fragment delimiter, invalid host/port syntax/ranges, and virtual-hosted bracketed IPv6. `internal/config/endpoint_test.go` checks literal/env loading, direct validation, formats/unwraps; `cmd/s3proxy/endpoint_test.go` checks all three commands fail before output/startup with zero backend connections.                          |
| ENDPOINT-01 AC2 — valid unavailable endpoints and base paths       | Pass   | Offline tests accept occupied/unhealthy/unavailable endpoints. Migrated `cmd/s3proxy/routes_test.go` and `internal/config/confidentiality_test.go` use valid escaped private paths. `internal/backend/s3/endpoint_test.go` verifies DNS/IPv4 modes and path-style IPv6 zones/mapped addresses; existing escaped-key signature tests pass.                                                                                           |
| ENDPOINT-01 AC3 — syntax-only contract                             | Pass   | Validation only inspects/parses URL fields and bounded decimal ports; it performs no resolution, binding, backend requests, or endpoint mutation. `README.md`, `website/docs/configuration.md`, and `docs/design.md` agree on migration, credentials outside userinfo, escaped base paths, optional ports, and IPv6 path-style addressing.                                                                                          |
| ENDPOINT-01 AC4 — checks                                           | Pass   | Config/CLI/backend/app packages pass in the final unit/race suites; no fixture was weakened to keep discarded URL components valid.                                                                                                                                                                                                                                                                                                 |
| CRED-01 AC1 — literal/env/optional backend-only field              | Pass   | `internal/config/load.go` and `internal/config/types.go` add the optional string only to backend credentials. `internal/config/session_token_test.go` verifies unchanged literal/env values through `Load`/`LoadFile`, absent/empty/unset behavior, independent target/environment snapshots, and rejection on inbound clients.                                                                                                     |
| CRED-01 AC2 — target authority and isolation                       | Pass   | `internal/backend/s3/client.go` filters inbound token header/query keys case-insensitively before signing a fresh request; copied target credentials supply the sole token. Direct captures alternate temporary/tokenless targets and preserve source requests; `internal/app/session_token_test.go` exercises real three-target fan-out under auth-none/header/presign.                                                            |
| CRED-01 AC3 — independent signing and composed workflow            | Pass   | `internal/testutil/s3signature/signature.go` computes canonical requests/HMAC without the SDK signer. `internal/backend/s3/session_token_test.go` supplies explicit expected signed headers and rejects token mutation/removal. Config-to-app HTTP captures also verify signatures, upload length/body, escaped paths, startup snapshots, and single-target reads.                                                                  |
| CRED-01 AC4 — confidential CLI/runtime diagnostics                 | Pass   | `cmd/s3proxy/routes.go` exposes an explicit topology DTO with no credentials. Config/CLI token-expression regressions and app startup/auth/dispatch/transport log checks omit sentinels. A non-HTTP-safe configured token fails with 502 before backend calls while retaining the header-name failure category.                                                                                                                     |
| CRED-01 AC5 — checks and lifecycle documentation                   | Pass   | Config/backend/HTTP/CLI/app checks pass. `README.md`, `website/docs/configuration.md`, `website/docs/operations.md`, and `docs/design.md` consistently describe operator-issued complete credentials, startup snapshots, replacement/restart before expiry, and no inbound session lifecycle, issuance, discovery, or refresh.                                                                                                      |
| CATALOG-01 AC1 — measured scaling and tradeoffs                    | Pass   | `internal/listbuckets/service_benchmark_test.go` matches the recorded 8/4,096-bucket, nil/wildcard/sparse/dense method; construction is outside timing and per-call lookup/copy/allocation is included. The recorded identical before/after runs show dense large-catalog medians of 26.653 → 0.477 ms; the 8-bucket sparse regression of 16.5 ns (10.1%) and allocation/range data are explicit. No new measurement is claimed.    |
| CATALOG-01 AC2 — semantics, ownership, concurrency                 | Pass   | `internal/listbuckets/service.go` sorts its own startup snapshot and returns independent storage. `internal/listbuckets/service_test.go` checks exact order/date, nil/wildcard/empty/explicit visibility, route-grant independence, input/principal/result mutation, and synchronized concurrent discovery. Required race verification passes.                                                                                      |
| CATALOG-01 AC3 — encapsulation                                     | Pass   | Catalog state remains a private service slice; explicit-policy lookup is call-local and principal state is not retained. No global, cross-package, or identity-keyed mutable cache was added.                                                                                                                                                                                                                                       |
| CATALOG-01 AC4 — checks                                            | Pass   | Catalog/HTTP suites and full vet/unit/race/build verification pass. `docs/design.md` accurately describes startup ordering, expected O(B + V) explicit discovery, and the separate HTTP `ListBuckets` operation-permission check.                                                                                                                                                                                                   |
| FINAL-07 AC1 — implementation acceptance and evidence              | Pass   | All 21 implementation acceptance bullets are covered above. Recorded test names/case coverage and benchmark method match inspected files; final checks leave no unresolved required verification or acceptance follow-up. Historical regression-first runs and benchmark samples remain attributed to implementation evidence.                                                                                                      |
| FINAL-07 AC2 — final checks                                        | Pass   | `make vet test`, `make test-race`, `make build`, and `git diff --check` pass. Untracked test/benchmark files also pass individual no-index whitespace checks.                                                                                                                                                                                                                                                                       |
| FINAL-07 AC3 — code/docs/test agreement                            | Pass   | Independently traced alternate request/authentication, offline CLI, selected-target/fan-out signing, and catalog ownership paths. Product contracts and test expectations agree; only the stale release-facing introduction needed clarification.                                                                                                                                                                                   |
| FINAL-07 AC4 — statuses and worktree discipline                    | Pass   | All six task statuses are completed after verification, dependencies are satisfied, and the historical findings remain intact. `CHANGELOG.md`, `.env`, and the pre-existing workflow edit retain their review-entry fingerprints. The subsequent parent audit is recorded in Coordinator Closeout.                                                                                                                                  |

Completion evidence:

- Independent review: read `AGENTS.md` and this entire task document, the complete
  intended tracked diff, all 11 untracked test/benchmark files, and the surrounding
  classifier, handler/auth filtering, executor/signature oracle, config diagnostics,
  CLI/app composition, fan-out, and catalog code/docs. No agents were launched.
- Findings: no implementation acceptance defect or independent follow-up was found.
  Changed only `docs/tasks/20261012-162534-gateway-workflow-health.md`: recorded this
  matrix/evidence and completed FINAL-07, made prior-task references repo-relative,
  and replaced the stale release-facing introduction with verified wording.
  Original findings, regression-first history, and benchmark measurements remain
  historical; the plan-creation baseline is superseded by the completion evidence,
  not a remaining gate.
- Required verification from `<repo-root>`, on Go 1.27.1 linux/amd64:
  - `make vet test` — exit 0; vet clean and all 19 unit-test packages passed
    (Go reused cached unit results).
  - `make test-race` — exit 0; all 19 packages passed with no race report; only
    `internal/listbuckets` reused a cached race result.
  - `ls -ld . dist` — inspected the build parent directories before `make build`.
    `make build` — exit 0; CGO-disabled host binary built at `dist/s3proxy` with
    `-trimpath -buildvcs=false`. Ignored build artifacts were not staged.
  - `git diff --check` — exit 0, no whitespace errors.
  - `git diff --exit-code -- CHANGELOG.md` — exit 0, no changelog diff.
- Additional review checks from `<repo-root>`: individual
  `git diff --no-index --check -- /dev/null <untracked-path>` checks passed for all
  11 new test/benchmark files and this task document. Review-entry/final
  `git hash-object .github/actions/setup-tools/action.yml CHANGELOG.md .env`
  fingerprints match. The workflow's pre-existing `github-token` addition and all
  intended implementation files were preserved; no staging or commits occurred.
- Benchmark judgement: the before/after method and results support the scoped
  optimization, including its small-policy cost. The existing measurements were
  reviewed rather than rerun; their single-host synthetic limitations remain valid.
  Required checks were not broadened/repeated after passing without a new concern.
- Limits: verification uses local HTTP fixtures and synthetic credentials; no live
  Docker/cloud stack or AWS STS issuance was run. External credential validity and
  expiry, live endpoint reachability, and end-to-end performance remain outside
  this plan's acceptance. No blocker remains for FINAL-07; the subsequent parent
  document audit is recorded in Coordinator Closeout.

## Release-Facing Notes

Verified during implementation and independently accepted in FINAL-07: stricter
unsupported operation-header rejection, safe authentication parse diagnostics,
endpoint validation compatibility changes, and optional backend session
credentials. `CHANGELOG.md` is excluded by request.

- HEADER-01 (verified): PUT requests carrying `X-Amz-Copy-Source` and S3 requests
  carrying `X-Amz-Multipart-Upload-Id` now reject header presence regardless of
  casing or empty, whitespace-only, or repeated values. Ordinary uploads must omit
  these headers entirely. Authentication filtering cannot erase the observed
  unsupported operation; supplied-signature and explicit payload-hash errors keep
  their existing precedence. Auth-none and successfully authenticated unsupported
  requests return `501 NotImplemented` before routing/dispatch/backend I/O.

- AUTHLOG-01 (verified): Malformed SigV4 header/presigned dates and invalid
  presigned expiry values now produce fixed, value-free authentication causes
  (`invalid X-Amz-Date` / `invalid X-Amz-Expires`). Parser inputs are absent from
  returned errors, unwrap chains, runtime logs, and XML responses. `errors.Is`
  classification, `403 AccessDenied`, request completion metadata, valid signature
  behavior, skew/seven-day/zero-expiry boundaries, and payload-verification order
  are preserved.

- ENDPOINT-01 (verified): Configuration now rejects endpoint userinfo, query
  components (including a bare `?`), fragments (including a bare `#`), opaque URLs,
  empty hostnames, and numeric ports outside `0`–`65535` or overflowing that range.
  Bracketed IPv6 endpoints require `force_path_style = true`. Remove formerly
  discarded URL components, correct invalid hosts/ports, and keep backend keys in
  the target's referenced `credential "static"` block. HTTP(S) DNS/IPv4 endpoints,
  optional ports, path-style IPv6 including zones/mapped addresses, and escaped
  base paths remain supported. Validation stays syntax-only and value-free;
  invalid configs fail before CLI success/topology or runtime logging, while
  unavailable endpoints still pass offline inspection.

- CRED-01 (verified): Backend `credential "static"` blocks now accept optional
  literal/env `session_token` values for externally issued temporary credentials.
  The selected target's token is included in outbound SigV4 signed headers;
  inbound header/presign tokens are stripped and cannot override or leak between
  destinations. Omitted/empty tokens preserve long-lived keys. Credential values
  remain outside topology, configuration diagnostics, and runtime logs. Credentials
  are a static startup snapshot: replace the complete set and restart before
  expiry. This adds no inbound client session authentication, STS issuance,
  ambient credential discovery, or automatic refresh.

## Consequential Deferrals And Limits

- Multipart, CopyObject, merged/pinned listing cursors, hot reload, IAM policy
  emulation, and automatic credential issuance/refresh remain the product's
  documented later-phase decisions. They require state/authorization/lifecycle
  designs; this plan does not silently expand their contracts.
- Metrics endpoints, global admission control, and native TLS are not introduced
  without deployment/backpressure requirements. Existing replay/transport bounds,
  external TLS, and log/process monitoring remain the operating model.
- No live AWS STS account is required or used to issue credentials here. Local
  independent signing tests establish token placement/signature behavior; actual
  cloud credential validity and expiry remain external operational prerequisites.
- No maintainer decision currently blocks this scoped plan.

## Coordinator Closeout

Completed: 2026-10-12 17:34:54 PDT (local audit)

- Read the entire final task file after all six sub-agent sessions finished.
  Confirmed six unique task IDs, six completed statuses, six completion-evidence
  sections, satisfied dependencies, and an independent acceptance matrix covering
  every implementation criterion. No required verification or follow-up is open.
- Execution was strictly sequential, with a new isolated session for each item:
  HEADER-01, AUTHLOG-01, ENDPOINT-01, CRED-01, CATALOG-01, then independent FINAL-07.
- Reviewed the production changes between agents and accepted FINAL-07's full
  code/test/documentation review and passing `make vet test`, `make test-race`,
  and `make build` evidence. Only task-document closeout wording changed afterward.
- Final `git status --short` matches the intended implementation, documentation,
  and new test files plus the original workflow edit. Independently confirmed
  `git diff --exit-code -- CHANGELOG.md` passes and inspected the preserved workflow
  `github-token` addition. No commits were made.
- Task paths/evidence are repo-scoped. Historical baseline failures, benchmark
  tradeoffs, deferred product decisions, and live-cloud/sandbox limitations remain
  explicit. The document's definition of done is satisfied.
- Follow-up phase: `docs/tasks/20261012-182925-runtime-lifecycle-portability-health.md`
  tracks newly reviewed runtime rewrite diagnostics, replay watcher lifetime, and
  cross-architecture hash selection; the completed outcomes above remain intact.
