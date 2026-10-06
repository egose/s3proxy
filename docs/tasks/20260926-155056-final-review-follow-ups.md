# Final Review: Redirect Boundaries And Diagnostic Confidentiality

Created: 2026-09-26 15:50:56 local time

## Objective And Scope

Close two independently actionable gaps confirmed by the fresh FINAL-01 audit of [S3 Workflow And Codebase Health](20260926-145643-s3-workflow-health.md). These were specified before implementation and completed in fresh sequential coordinator-assigned sessions. The final reviewer did not implement these tasks or delegate them. Original FINAL-01 is now completed after independent reaudit and fresh final verification; see [its resumed completion evidence](20260926-145643-s3-workflow-health.md#resumed-final-01-completion-evidence). Earlier pending/blocked statements in the execution records below are historical and superseded by the final confirmation.

Preserve existing work and sandbox resources, read `AGENTS.md`, and do not commit or modify `CHANGELOG.md`. Document externally visible corrections in README/design/website. Use the isolated sandbox wrapper and preserve unfamiliar resources for any live check. No broad auth, routing, or config rewrites are requested.

## Confirmed Review Evidence

- Temporary local review probes exercised the actual `s3.NewClient` and `config.Load`, not just standard-library behavior. The probes were removed after recording their results; they did not change implementation or contact external services.
- `go test -count=1 -run '^TestFinalReview' ./internal/config ./internal/backend/s3` failed the desired confidentiality/operation-boundary assertions. A 307 from `/bucket/key` to `/bucket` caused an actual second `DELETE /bucket` call and the executor returned success. A malformed endpoint loaded through `env()` exposed its sentinel password in the returned config error.
- The corrected timeout probe (replace the existing `timeout = "5s"`, rather than adding a duplicate attribute) also failed: `go test -count=1 -run '^TestFinalReviewEnvCompileDiagnosticProbe$' ./internal/config` returned `invalid timeout: time: invalid duration "FINAL_REVIEW_SENTINEL_SECRET"`. The endpoint probe returned `invalid endpoint: parse "https://user:FINAL_REVIEW_SENTINEL_SECRET@example.test/%GG": invalid URL escape "%GG"`.
- These are distinct from the resolved rewrite/signing and literal-HCL defects. Redirects create subsequent requests after validation; field-specific compilation creates diagnostic strings after literal HCL evaluation. Existing tests do not exercise these paths.

### Task REDIRECT-01: Prevent Automatic Upstream Requests Outside The Validated Operation

Status: completed

Execution record:

- Fresh isolated sequential REDIRECT-01 session: read both task plans, repository `AGENTS.md`, and the requested project task-as-you-go skill; confirmed REWRITE-01/SIGN-01 completed and inventoried pre-existing dirty/untracked work. Implementing regression-first at the shared backend boundary.
- Regression-first: `go test -count=1 -run 'TestClientRejectsRedirect|TestBuildRejectsUpstreamRedirects' ./internal/backend/s3 ./internal/app` — FAIL before production changes. All 60 direct HTTP cases issued a second same-host or receiver request; permissive callbacks ran. Tracked malformed Locations leaked sentinel data through wrapped errors, and 307/308 created a second replay body. The app fixture initially missed its route because `path_prefix = "/"` is strict; corrected it to `/bucket`, then `go test -count=1 -run '^TestBuildRejectsUpstreamRedirects$/307/DELETE/bucket$' ./internal/app` — FAIL with two upstream requests and successful execution instead of controlled failure.
- Additional regression-first: `go test -count=1 -run '^TestHandlerRedirectReleasesReplayBudget$' ./internal/httpapi` — FAIL in all ten cases before production changes, with two requests per configured destination for single/fan-out PUT. Replay cleanup itself already passed and is preserved.
- Material boundary finding: net/http parses Location before `CheckRedirect`, and prepares 307/308 replay bodies before that callback. Enforcement therefore wraps the copied client's transport to reject redirect statuses before either step; the copied refusal callback cannot be replaced by the caller's permissive policy. The wrapper closes bodies once and returns a fixed cause without URL/body/Close-error details. Target cancellation uses the existing backend error path.

Kind: defect

Priority: P1 — automatic redirects bypass the shared outbound method/bucket/key check.

Suggested agent: fresh outbound HTTP/S3 boundary specialist

Dependencies: REWRITE-01 and SIGN-01 in the main plan (completed implementation sessions)

Primary ownership: `internal/backend/s3/client.go`, backend boundary/signing tests, focused real-handler HTTP tests, contract docs. Inspect `internal/app/app.go` composition, but enforce policy for direct executor callers too.

Finding: `client.Do` validates the initial request, then calls the injected `http.Client.Do` with its default redirect behavior. App composition supplies no `CheckRedirect`. A 307/308 can preserve DELETE while redirecting its object path to the bucket root; 301/302/303 can also change methods. No redirected request is reclassified or re-signed. A conforming authenticated S3 backend should reject the stale path signature, so the review does **not** claim a demonstrated signed DeleteBucket exploit against MinIO/AWS. The confirmed violation is an unvalidated subsequent upstream request, and successful execution against a permissive local backend; signed headers/body can also be forwarded according to the HTTP client's redirect rules.

References:

- `internal/backend/s3/client.go`: `NewClient`, `client.Do`, initial shape/classification checks and `c.httpClient.Do(outReq)`.
- `internal/app/app.go`: `Build`, construction of `http.Client` with only `Transport`.
- `internal/backend/s3/operation_boundary_test.go`: rejects initial malformed shapes but has no redirect cases.

Requirements:

1. Enforce a no-automatic-upstream-redirect policy at the shared backend client boundary for both composed and direct callers. Copy any supplied client before changing redirect policy; preserve its transport, jar, timeout and caller ownership. Do not permit a caller-supplied redirect callback to bypass the invariant.
2. Return a controlled proxy/backend error for a redirect response rather than forwarding a redirect Location that sends the inbound client outside its virtual namespace. Close the redirect response body, release target cancellation, and preserve ordinary non-redirect error-body handling. Do not implement S3 region discovery or re-sign arbitrary redirect destinations in this task.
3. Cover 301/302/303/307/308 across representative GET/PUT/DELETE paths, same-host bucket-root and cross-host redirects, and body replay. Preserve explicit Content-Length and ordinary signatures. Avoid logging redirect Location credentials/query/key data.
4. Document that operators configure the correct backend endpoint/region and redirects are rejected; define the controlled HTTP/S3 error through existing proxy error handling.

Acceptance criteria:

- A validated DeleteObject receiving a 307/308 to the bucket root never issues `DELETE /bucket`; a 301/302/303 never issues a follow-up GET. Cross-host receivers see zero calls and no signed headers/body.
- Restricted-principal real-handler and direct-executor tests return controlled failure with no second upstream request. Tests demonstrate caller-owned `http.Client` remains unchanged and a permissive custom redirect callback cannot bypass the backend policy.
- Tracked redirect bodies close exactly once; request replay budget and target cancellation are released; logs/client XML do not expose sentinel redirect URL secrets. Ordinary successful object and listing/error responses retain their contracts.
- Regression-first failure is recorded; `go test ./internal/backend/s3 ./internal/httpapi ./internal/app`, `make vet test`, and `make test-race` pass.

Verification: all commands from `<repo-root>`; above focused/full commands, `git diff --check`, unchanged CHANGELOG. Original FINAL-01 must repeat its required final checks after both follow-ups.

Completion evidence:

- Changed implementation: `internal/backend/s3/client.go` only. `NewClient` owns a shallow client copy, keeps timeout/jar and delegates through the supplied/default transport, and installs redirect refusal independent of caller callbacks. The transport rejects exactly 301/302/303/307/308 before net/http redirect parsing/replay, closes the response once (including Close-error cases), and returns a fixed safe backend error; existing executor cancellation and proxy `502 InternalError` handling complete cleanup. Inspected `internal/app/app.go`: its existing composition uses this shared constructor and requires no production app change.
- Changed regressions: new `internal/backend/s3/redirect_test.go`, `internal/httpapi/redirect_test.go`, `internal/app/redirect_test.go` (137 subcases). Sixty direct HTTP cases cover every redirect status across GET/PUT/DELETE, bucket-root/cross-host receivers, default transport and permissive caller policies. Initial PUT bytes, explicit Content-Length and independent S3 HMAC signatures are preserved; no second request or caller redirect callback runs. Client identity/settings stay caller-owned, timeout/jar are retained, and direct callers retain/release their replay reservation normally.
- Fifteen tracked-body cases cover valid, malformed and absent Location for all five statuses: zero response reads, one Close even when Close returns a sentinel error, one initial replay-body acquisition, no redirect replay, no sentinel in any wrapped error, and prompt target-context cancellation with a one-hour target deadline and no HTTP-client timeout masking that check. Seven non-redirect cases preserve 200/201/204/304/403/404/500 headers/status/body streaming and closure. Existing object/signing/listing/error-body contracts pass in package/full suites.
- Forty-five real HTTP app-composition cases load HCL and use the real authenticator, single-operation-restricted principal, authorizer, router, rewriter, dispatcher and backend client. Every signed GET/PUT/DELETE redirect yields `502 InternalError`, no Location, one upstream call, zero receiver calls, no sentinel URL/body data in client XML or logs, and failure/completion telemetry. Ten real-handler PUT cases prove both single-target and two-target fan-out failures release the active replay budget to zero and issue only the configured initial requests.
- Changed docs: `README.md`, `docs/design.md`, `website/docs/api-reference.md`, and this task's execution/status/evidence. Documented correct endpoint/region configuration, rejected statuses, safe failure and resource ownership, why CheckRedirect alone is insufficient, ordinary 304/error streaming, configured ordered-failover behavior and existing primary-error precedence.
- Verified from `<repo-root>`: `go test -count=1 -run 'TestClientRejectsRedirect|TestBuildRejectsUpstreamRedirects|TestHandlerRedirectReleasesReplayBudget' ./internal/backend/s3 ./internal/app ./internal/httpapi` — PASS after the fix; `go test ./internal/backend/s3 ./internal/httpapi ./internal/app` — PASS; `make vet test` — PASS (vet and all 17 unit packages, including final strengthened timeout/ownership assertions); `make test-race` — PASS (all 17 packages). `gofmt -l internal/backend/s3/client.go internal/backend/s3/redirect_test.go internal/httpapi/redirect_test.go internal/app/redirect_test.go` — PASS, empty output; `git diff --check`, `git diff --exit-code -- CHANGELOG.md`, and `git diff --cached --exit-code -- CHANGELOG.md` — PASS, empty output.
- Scope/result: all REDIRECT-01 acceptance checks pass. Prior work preserved; no subagents, commits or CHANGELOG edits. No unresolved REDIRECT-01 finding remains. CONFIG-02 is still pending and original FINAL-01 is still blocked, with its statuses and independent final/live verification left to their assigned sessions.

### Task CONFIG-02: Keep Environment-Derived Values Out Of Field Compilation Diagnostics

Status: completed

Execution record:

- Fresh isolated sequential CONFIG-02 session: read follow-up/main plans, repository AGENTS.md and the requested project task-as-you-go skill; confirmed CONFIG-01 and REDIRECT-01 completed, inventoried pre-existing dirty/untracked work, and preserved it. Regression-first implementation is scoped to configuration diagnostic confidentiality and its CLI/docs coverage. FINAL-01 remains blocked for independent reaudit.
- Regression-first: `go test -count=1 -run 'TestEnvDiagnosticConfidentiality|TestEnvCompileDiagnosticSourceLocation|TestValidateCLIEnvDiagnosticConfidentiality' ./internal/config ./cmd/s3proxy` — FAIL before production changes. Confirmed endpoint userinfo/escape/port, target and four listener durations, unknown duration unit, regex capture-name, template-function, credential/parser/destination/bucket/policy refs (including stripped suffixes), policy/route operations and auth/route enums expose env-derived data. HCL duplicate-object-key errors also expose evaluated keys. All six real CLI subprocess cases (endpoint/duration/template/ref/enum/HCL) failed confidentiality assertions. Tests report only case identities, never offending diagnostics/values.
- Inspection/design decision: all environment-derived values are sensitive. HCL diagnostic subject ranges identify enclosing attributes; expressions containing any `env()` call withhold value-bearing Detail while preserving safe HCL Summary, original Subject, and source block/field identity. Later compilation/validation never echo attribute values, including literal values, eliminating dependence on matching secrets after quoting, parser excerpts or reference stripping. Structured URL/regex causes and fixed duration/template categories replace raw wrapped parser errors. Existing literal-value semantics and original source bytes are preserved. Public filename/block labels remain diagnostic metadata.
- Fixture/compatibility corrections during development: the initial template-token fixture was legal Go-template syntax; changed it to an invalid token. A duplicate-visible-name fixture initially duplicated its route attribute; corrected it to two valid blocks. Existing public-reference/duplicate tests expected value echoes; updated nine assertions to the new value-free field/block/cause contract. The first targeted rerun passed; the first package run exposed only those expected diagnostic-contract assertion changes. Namespace listing validation was already value-free; added key_template field context to its existing safe causes.

Kind: defect

Priority: P2 — validation/CLI errors can expose environment secrets despite safe literal HCL evaluation.

Suggested agent: fresh HCL diagnostics and configuration security specialist

Dependencies: CONFIG-01 in the main plan (completed implementation session); execute after REDIRECT-01 to keep final integration sequential.

Primary ownership: `internal/config/load.go`, `internal/config/validate.go` as needed, config/CLI diagnostic regressions, configuration docs.

Finding: native `env()` safely returns literal strings, but `compile` wraps `url.Parse`, duration, regexp and template errors and interpolates some invalid refs/enums. The endpoint and duration cases above are confirmed leaks. Existing `TestLoadFile_EnvErrorsDoNotExposeSecrets` exercises HCL parse/decode and missing credentials, not those later compilation diagnostics. CLI `validate` returns the app/config error, so the same data can reach command output. Regex/template/reference cases need bounded inspection before claiming additional confirmed leaks.

References:

- `internal/config/load.go`: `Load`, `compile` endpoint/timeout/regex/template error construction, `parseTimeouts`, `parseOptionalDuration`, `resolveCredentialRef`.
- `internal/config/validate.go`: field/ref/enum diagnostics.
- `internal/config/env_test.go`: `TestLoadFile_EnvErrorsDoNotExposeSecrets`.
- `cmd/s3proxy/main_test.go`: `TestValidateCommandEnvironmentValues`; `internal/app/app.go`: `ValidateConfig`.

Requirements:

1. Preserve literal env evaluation, ordinary field validation, unset-variable behavior and original HCL diagnostic positions. Establish a consistent confidentiality boundary for environment-derived values used by string fields, including later compilation/validation errors; do not merely redact a single sentinel or one endpoint spelling.
2. Ensure malformed endpoint userinfo/path/query and invalid duration values cannot leak their environment text through returned errors, wrapped errors, or CLI stdout/stderr. Keep useful phase/field/block identity and safe diagnostic causes/source locations. Decide and document whether all env-derived values are treated as sensitive; do not label only credential fields sensitive while arbitrary env strings can reach unsafe errors.
3. Inspect regex/template/ref/enum error paths for the same root cause and cover confirmed cases in this task. Avoid printing environment contents during tests or debugging. Preserve values containing quotes, newlines, backslashes, Unicode and template markers exactly on successful loads.
4. Add regressions before implementation for direct `Load`/`LoadFile` and real validate-command errors, including the two confirmed probes above and transformed/quoted value diagnostics where relevant. Preserve ordinary non-secret error usability and all published config examples.

Acceptance criteria:

- An env endpoint value `https://user:<sentinel>@example.test/%GG` fails controllably without the password/URL data in errors or CLI output; an env timeout `<sentinel>` reports invalid timeout without disclosing the value.
- Confirmed additional field-specific leakage paths are covered by negative tests; successful literal-value and exact original-source-location regressions still pass. Missing variables and normal public configs remain compatible.
- Diagnostics retain the relevant field/block and safe cause rather than silently succeeding or reporting a generic unrelated error. No raw env values are persisted in logs/test artifacts.
- Regression-first evidence and inspected-path findings are recorded; `go test ./internal/config ./internal/app ./cmd/s3proxy`, `make vet test`, and `make test-race` pass.

Verification: all commands from `<repo-root>`; above focused/full commands, `git diff --check`, unchanged CHANGELOG. Then return to original FINAL-01 for independent acceptance audit and final local/live verification.

Completion evidence:

- Changed implementation: `internal/config/load.go`, `internal/config/validate.go`, new `internal/config/diagnostics.go`. Load errors now identify compile/validate phase and filename. HCL errors use original attribute/source provenance rather than secret matching; evaluated-value Detail is withheld for any expression containing `env()`, while safe Summary, Subject range, and block/field context remain. Unrelated public HCL details are retained. Compilation never returns/wraps raw URL, duration, regex or Go-template parser errors; safe category messages preserve the cause and original expression range. All compiler/validator attribute-value interpolation was removed, covering refs, normalized ref suffixes, enum/operation values, and duplicate visible names. Direct `Validate` uses the same value-free field/rule messages. Source-literal labels remain useful identifiers.
- Changed tests: new `internal/config/confidentiality_test.go` and `cmd/s3proxy/confidentiality_test.go`; updated `internal/config/load_test.go` for the intentional value-free contract and `internal/config/env_test.go` to use non-value-bearing literal-test names. Thirty-two negative configurations each execute both `Load` and `LoadFile` (64 executions), checking returned errors and every unwrap with ordinary, detailed, and Go-syntax formatting. Coverage includes malformed endpoint userinfo/path/query/host/port; target and every listener timeout; quoted/escaped/Unicode values; duration-unit extraction; regex capture names; Go-template function/token failures; credential/parser/destination/bucket/policy refs and stripped suffixes; auth/route enums and operations; duplicate visible names; HCL object conversions, invalid indices, nested env calls, duplicate keys and interpolated duplicate keys. Assertions fail without printing captured environment data.
- Additional acceptance evidence: nine direct `Validate` mutations verify field/cause confidentiality independently of loading. Exact original compile positions after leading LF/CRLF blanks pass for timeout (`47,22`), endpoint (`44,22`), regex (`56,13`) and listener read (`16,19`); the new env decode test retains `5,24` with block/field identity, and the public numeric decode test retains `5,25` plus its detailed number-type cause. Existing CONFIG-01 exact parse/decode locations, missing/empty-variable behavior, five literal values across five expression forms, and ten published/sandbox configuration examples all pass. A new success case checks compiled env endpoint/duration/regex/template/ref/enum fields and exact literal prefix/template values, then revalidates the Runtime.
- Real CLI evidence: the new subprocess suite builds the actual `cmd/s3proxy` binary into a test temporary directory and invokes `validate --config` through `main`/Cobra/app/config. Eight invalid cases (endpoint, target/listener duration, template, regex, reference, enum and HCL duplicate key) exit 1 with empty stdout and useful safe stderr, excluding full/partial sensitive URL and quoted/transformed value data. The ninth subprocess case preserves literal credential contents containing HCL markers, quotes, newline, backslash and Unicode, exits 0, and emits exactly `config is valid\n` with empty stderr. Existing command-object/app validation tests also pass. Temporary config fixtures contain env calls rather than evaluated values.
- Inspected paths: all error construction in `load.go` and `validate.go`, HCL's pinned `gohcl.DecodeExpression` conversion diagnostics and syntax summaries, `namespace.ValidateTemplate`/`RawTemplatePrefix`/`New`, and CLI/app validation propagation. Namespace mapping errors already return fixed causes without template/prefix contents; its validator now adds field context. HCL object conversion/index/nested-call cases were already confidential but lacked explicit field/block context; duplicate-object-key details were a confirmed additional leak. Labels/types are parsed source literals, not env-derived fields. No additional independently actionable CONFIG-02 finding remains.
- Changed docs: `README.md`, `docs/design.md`, `website/docs/configuration.md`, and this follow-up's status/execution/evidence. Documented all-env sensitivity, value-free compilation/validation including literal inputs, provenance-based HCL handling, preserved values/source locations, safe causes, public filename/label metadata and the intentional diagnostic contract change.
- Verified from `<repo-root>`: `go test -count=1 -run 'TestEnvDiagnosticConfidentiality|TestEnvCompileDiagnosticSourceLocation|TestValidateCLIEnvDiagnosticConfidentiality' ./internal/config ./cmd/s3proxy` — FAIL before fixes as recorded above, PASS after the initial implementation; `go test ./internal/config ./internal/app ./cmd/s3proxy` — PASS after completed regressions; `make vet test` — PASS (vet and all 17 unit packages, including the final phase wrappers); `make test-race` — PASS (all 17 packages). `gofmt -l internal/config/load.go internal/config/validate.go internal/config/diagnostics.go internal/config/load_test.go internal/config/env_test.go internal/config/confidentiality_test.go cmd/s3proxy/confidentiality_test.go` — PASS, empty output. `git diff --check`, `git diff --exit-code -- CHANGELOG.md`, and `git diff --cached --exit-code -- CHANGELOG.md` — PASS, empty output.
- Scope/status/limitations: CONFIG-02 acceptance and all required verification pass. Prior work preserved; no commits, CHANGELOG edits or subagents. Diagnostic confidentiality covers environment-derived configuration attributes; config filenames and literal block labels remain public, and returned successful Runtime values remain intact. No live-stack/release check was run in this configuration-only session. Original FINAL-01 stays **blocked** until a fresh independent reviewer reaudits both follow-ups and repeats its required final local/live verification; this implementation session does not grant final sign-off.

## Definition Of Done

Both follow-ups complete in fresh coordinator-assigned sessions with changed-file and exact-command evidence; original FINAL-01 independently verifies the corrected alternate paths, repeats required checks, and updates its blocked status only after all acceptance criteria pass. No live resource migration or restoration is authorized by these tasks.

Definition of Done confirmation (resumed independent FINAL-01, 2026-09-26): **met**.

- All four acceptance criteria for each follow-up were independently checked against production code, tests and public docs, including net/http's Location/replay ordering and HCL diagnostic provenance. The original REWRITE-01 operation-boundary and CONFIG-01 confidentiality reservations are resolved. Both tasks remain completed; FINAL-01 is completed. No new follow-up or acceptance blocker remains.
- After both implementations: `make vet test`, `GOFLAGS=-count=1 make test-race` (all 17 packages uncached), `make build`, `go test -tags integration -run '^$' ./internal/integration`, sandbox syntax/shellcheck/fake lifecycle checks, `git diff --check`, and `git diff --exit-code HEAD -- CHANGELOG.md` — PASS.
- `SANDBOX_PROJECT_NAME=s3proxy-final-20260926-162604 make sandbox-integration-up` — PASS, all 29 top-level tests and exit 0. Preflight/postcheck assertions preserved all 27 pre-existing containers with their names/running states, 301 volumes and eight networks; selected-project/proxy teardown passed. New test volumes were retained, with no migration or restoration attempted.
- Full criterion-by-criterion evidence, final nine-task status audit and remaining documented limitations are in [FINAL-01's resumed completion record](20260926-145643-s3-workflow-health.md#resumed-final-01-completion-evidence); sandbox preservation is cross-recorded in [SANDBOX-01](20260926-151017-sandbox-compose-isolation.md). No resumed-review source changes, commits, CHANGELOG edits or subagents were needed. This confirmation supersedes the historical statements that CONFIG-02 was pending or FINAL-01 blocked.
