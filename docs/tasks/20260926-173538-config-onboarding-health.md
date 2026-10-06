# Configuration Validation And Binary-Only Onboarding

Created: 2026-09-26 17:35:38 (local timestamp).

## Objective And Analysis

Continue the [completed upload-boundary phase](20260926-172011-streaming-upload-boundary.md) with configuration/deployment usability. The product is a single-binary virtual S3 gateway: operators should catch malformed listener syntax before rollout, and an installed binary can usefully provide a version-matched authenticated starter without a source checkout.

Reviewed config validation, app startup, CLI, onboarding/public examples, Docker packaging and relevant completed task records. The read-only discovery session confirmed the address defect with the existing `dist/s3proxy` (`079d2aa-dirty`) using synthetic stdin configurations: `127.0.0.1`, `:65536`, and `http://127.0.0.1:8080` each passed `validate`/`routes`, then failed parsing in `serve` before successful binding. No live backend or stack was touched. Current source corroborates that address validation only checks nonempty.

`print-example-config` is an **optional improvement**, explicitly proposed in `docs/design.md`; README/quickstart already contain examples. The benefit is offline, version-matched binary onboarding, not filling an absent documentation capability. The user has asked to continue reviewing and implementing actionable improvements, so it is included as a bounded follow-up.

## Scope, Deduplication, And Working Rules

- Prior config tasks cover duplicate/ref/policy rules, literal env evaluation and safe diagnostics, not listener syntax. Completed CLI-02 adds routes inspection, not starter output. Its test using a malformed address as evidence of offline behavior needs correction: syntactic validity and local bind availability are different properties.
- Baseline `make vet test` passes (19 cached unit packages) before creation. No comprehensive dependency, release, cloud or container-runtime audit is claimed.
- Three tasks, in order, each in a fresh separate sub-agent session; no nested agents. Mark `in_progress` before work and `completed` only after acceptance/checks with `Completion evidence`. The final reviewer must not be either implementer.
- P2 = definite predeployment validation gap; P3 = optional onboarding convenience; final review is the completion gate. Shared CLI/docs files are serialized.
- Commands run from `<repo-root>`. Preserve all existing dirty/untracked user/prior-agent work. No commits/resets/whole-file restoration, broad formatting or live stack operations. Follow `AGENTS.md` and use `apply_patch` for edits.
- **Never edit `CHANGELOG.md`.** Starting SHA-256: `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`. Record externally visible corrections here and in relevant public docs.
- No interactive setup, credential generation, file-writing command option, DNS/bind health checks, Docker entrypoint redesign, or new S3 APIs. Runtime availability and backend provisioning remain operator responsibilities.

### Task LISTENER-01: Reject Malformed Listener Addresses During Validation

Status: completed

Kind: defect

Priority: P2 — invalid rollout inputs pass the documented predeployment gate.

Suggested agent: configuration validation/CLI contract implementer.

Dependencies: none.

Primary ownership: `internal/config/validate.go`, focused config/CLI tests including `cmd/s3proxy/routes_test.go`, listener configuration/deployment/design documentation.

Finding: `validateListener` only checks empty address, while `App.Run` delegates to `net.Listen`. Missing port separators, URL-form addresses and out-of-range numeric ports pass both validation and topology reporting but fail at startup. Existing routes test calls a malformed address “non-bindable” to prove no network side effects; retain that guarantee with a syntactically valid unavailable address instead.

References:

- `internal/config/validate.go:37-59` (`validateListener`).
- `internal/app/app.go` (`ValidateConfig`, `Run` and `net.Listen`).
- `cmd/s3proxy/routes_test.go:101-108` (non-bindable listener fixture).
- `website/docs/deployment.md` (validate before rollout); `website/docs/configuration.md` (listener fields).

Requirements:

1. Validate structural TCP host:port syntax and numeric port range at the existing shared config boundary, with safe field/block errors that omit raw address and parser excerpts. Reject the three reproduced examples and analogous structural malformed forms.
2. Keep validation offline: no DNS, service lookup, interface ownership checks or listener binding. Preserve wildcard host, IPv4, bracketed IPv6 including zones, hostnames and numeric zero/ephemeral behavior. Inspect Go's existing parser contract and preserve empty-port behavior and named service ports rather than accidentally imposing numeric-only ports. Document that service existence/resolution, hostname resolution, address ownership and port availability are runtime checks. Do not invent a broad DNS naming policy.
3. Use literal/env secret sentinels in invalid-address tests; errors for direct Load/Validate and real CLI `validate`/`routes`/`serve` must remain value-free and useful. Invalid input fails before success output, topology or startup logging. Preserve public labels as the existing diagnostic contract allows.
4. Replace the malformed offline routes success fixture with a syntactically valid unavailable local address; retain occupied-listener success. Valid but unavailable addresses still pass offline validation and route reporting without backend calls.
5. Add regression-first tests that fail before the fix and pass after. Update docs to distinguish syntax validation from actual ability to bind; note the intentional earlier rejection of previously accepted malformed configurations.

Acceptance criteria:

- Missing port separator, URL syntax and out-of-range numeric ports fail config loading/direct validation and real CLI commands with no partial success output or raw values.
- Boundary numeric ports, wildcard/IPv4/IPv6/zone/hostname/named-service/empty-port forms preserve the documented Go-compatible contract; valid occupied/unavailable addresses pass offline commands.
- No network/bind lookup is introduced into config validation or routes inspection; pre-existing diagnostic confidentiality and public-example tests pass.
- Docs/test fixtures accurately reflect the new syntax gate and remaining runtime checks.

Verification: regression-first focused tests; `go test -count=1 ./internal/config ./internal/app ./cmd/s3proxy`; `make vet test`.

Completion evidence:

- Changed: `internal/config/validate.go`; new `internal/config/listener_address_test.go` and `cmd/s3proxy/listener_address_test.go`; the unavailable-listener fixture in `cmd/s3proxy/routes_test.go`; listener/rollout rationale in `website/docs/configuration.md`, `website/docs/deployment.md`, and `docs/design.md`; this record.
- Shared enforcement: `config.Validate` now checks `net.SplitHostPort` structure, rejects URL-form input (including URLs without an explicit port), and checks numeric port range without resolving or binding. Errors preserve the public listener label and `address` field with fixed causes; raw parser errors/addresses are never wrapped. Existing Load/LoadFile/app/CLI callers inherit the boundary before success/topology output or startup logging.
- Go contract inspected in Go 1.26.6 `src/net/ipsock.go` (`SplitHostPort`, `internetAddrList`), `port.go` (`parsePort`), and `lookup.go` (`Resolver.LookupPort`). Preserved wildcard/IPv4/hostname/bracketed IPv6 and zones, decimal `0`–`65535`, leading zeros, optional signs, negative zero, empty-port zero, and Go's bare `+`/`-` zero behavior. Negative nonzero and overflowing numeric ports fail safely. Nonnumeric service names remain runtime inputs; no numeric-only service rule or DNS naming policy was added. Runtime still determines service/hostname resolution, zone/interface existence, address ownership, permissions, and port availability.
- Regression-first: before implementation, `go test -count=1 ./internal/config ./cmd/s3proxy -run 'TestListenerAddress'` failed for malformed direct Validate/Load inputs and all 48 negative real-CLI combinations (8 cases × literal/env × validate/routes/serve). CLI validate/routes incorrectly succeeded; serve exposed address/parser details and config-loaded logging. Positive offline controls passed. Corrected test-only assumptions about Load's existing wrapper, safe filenames (Go's subtest-derived temp paths initially contained sentinel words), and the expected fixed error phrase; the empty-address existing behavior was retained. A second pre-fix direct-config run, `go test -count=1 ./internal/config -run 'TestListenerAddressInvalid/(empty|missing_port|URL|high_port)$'`, confirmed the malformed-input failures after the wrapper correction.
- After: `go test -count=1 ./internal/config ./cmd/s3proxy -run 'TestListenerAddress|TestRoutesCLI'` passed. Direct Validate plus literal/env Load and LoadFile cover missing separators, URLs, numeric bounds/overflow, bracket errors, unbracketed IPv6, and extra ports; sentinel diagnostics are checked through wrapped errors. All 48 real-CLI negative combinations exit 1 with empty stdout, useful safe diagnostics, and no startup logs or sentinel values. Positive offline commands cover occupied listeners, `192.0.2.1:8080`, unresolved `.invalid` hosts/nonexistent services/zones, boundaries, wildcards, IPv6, signs and empty ports, with exact success/topology output and zero backend connections. Existing routes occupied-port coverage remains; its formerly malformed non-bindable fixture is now `192.0.2.1:8080`.
- Required checks passed: `go test -count=1 ./internal/config ./internal/app ./cmd/s3proxy` (including existing confidentiality and public-example tests); `make vet test` (all 19 unit packages). Scoped `gofmt -l` reported no files; scoped whitespace checks passed. CLI tests build temporary actual executables; no installed/dist artifact or live stack was used.
- Preservation: edits used `apply_patch`; existing dirty/untracked work retained. `CHANGELOG.md` SHA-256 remains `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`. No commits, resets, broad formatting, nested agents, or live Docker/backend operations. No LISTENER-01 blocker remains. EXAMPLE-01 and FINAL-05 remain pending for their separate sessions.

### Task EXAMPLE-01: Print A Static Authenticated Starter Configuration

Status: completed

Session: isolated EXAMPLE-01 implementation; LISTENER-01 completion/evidence confirmed before starting.

Kind: improvement

Priority: P3 — convenient offline onboarding for users with only the binary installed.

Suggested agent: CLI/onboarding implementer.

Dependencies: LISTENER-01.

Primary ownership: `cmd/s3proxy`, static example stored with the command if appropriate, focused CLI tests, `README.md`, `docs/design.md`, `website/docs/quickstart.md` and relevant deployment/operations guidance.

Finding: main registers serve/validate/routes/version. Design proposes `print-example-config`, but installed-binary users must currently consult docs/source for HCL. Existing online examples remain useful; this feature should provide one predictable starter without expanding configuration state or credential management.

References:

- `cmd/s3proxy/main.go` (registration); `cmd/s3proxy/routes.go` (output-error handling pattern).
- `docs/design.md` (optional future CLI commands).
- `website/docs/quickstart.md` and `README.md` (existing starter workflows).
- `internal/config/public_examples_test.go` (example validation contract).
- `Dockerfile:30-36` (serve-specific entrypoint); publish workflow uses entrypoint override for CLI commands.

Requirements:

1. Add no-argument `s3proxy print-example-config` writing one static HCL template to stdout with deterministic bytes and a trailing newline. Printing must not read/evaluate environment values, load config, generate secrets, write files, bind or contact backends. Positional args/unknown flags fail; propagate write/short-write errors.
2. Starter uses a native loopback address (e.g. 127.0.0.1:8080), path-style addressing, sigv4_static authentication, separate inbound/backend credential env references, explicit narrow route/operation permissions, one target/visible bucket and a valid route/rewrite mapping supporting ordinary object lifecycle, HeadBucket and ListObjectsV2; include client ListBuckets permission for virtual discovery.
3. Reference endpoint/credentials using clear documented environment names; choose a documented region and pre-existing backend bucket convention. Avoid plaintext credential literals. Set explicit bounded replay settings and sensible upload/target timeout defaults, explaining deployment-dependent adjustments instead of implying defaults fit all object sizes.
4. Output contains literal `env("...")` calls, never substituted values. Tests run real CLI with required variables unset, valid, and secret-filled/malformed, requiring byte-identical output and empty stderr. Printing must succeed without configuration prerequisites; later validation must fail safely when required vars are missing.
5. Test the actual generated output through the real `validate` and `routes` commands with documented synthetic variables, asserting expected topology and access/listing behavior at the configuration contract. Use counting listeners or existing fixture patterns to establish offline behavior. Do not start a live stack.
6. Document print-to-file shell usage, variable setup, backend bucket prerequisite, validate/routes/serve workflow, and separate client credentials. Keep the existing manual quickstart available but clearly distinguish its auth-none configuration from this authenticated starter.
7. Document Docker usage via `--entrypoint /usr/local/bin/s3proxy` without changing the image entrypoint. Explain that the native loopback listener must be changed to `:8080` inside the container for published ports, with restricted host publishing and existing TLS guidance. Do not claim an unrun Docker command was verified live.

Acceptance criteria:

- Output is version-owned static HCL, deterministic independent of environment, with no secret values/runtime logs/side effects; command argument and output errors return nonzero.
- Generated HCL validates with documented synthetic vars and yields exactly the documented single-route topology; missing prerequisites fail subsequent validation safely. Auth/visible-bucket/operation permissions are explicit and compatible with intended object/list/head/discovery workflows.
- Tests prove no environment disclosure and no listener/backend use, including arbitrary environment content; existing CLI/config/public-example tests pass.
- Native and Docker onboarding docs match actual defaults, environment names, pre-existing bucket requirements, supported operations and command registration. Design no longer describes the delivered command as future work.

Verification: `go test -count=1 ./cmd/s3proxy ./internal/config ./internal/app`; `make vet test`; actual local CLI print → validate → routes workflow with synthetic values (may be through subprocess tests).

Completion evidence:

- Changed: registration in `cmd/s3proxy/main.go`; new `cmd/s3proxy/example.go` and `cmd/s3proxy/example_test.go`; `README.md`, `website/docs/quickstart.md`, `website/docs/deployment.md`, `docs/design.md`; this record. LISTENER-01 was confirmed completed before this isolated session marked EXAMPLE-01 in progress.
- Command boundary: no-argument `print-example-config` writes a Go-owned static HCL constant with a trailing newline via the command output writer. The execution path has no config/app/environment/network/file-management dependency. Positional args, unknown flags (including config/output flags), write failures, partial failed writes and silent short writes fail. Real `/dev/full` output failure exits 1 with `write example config` diagnostics; unit writers verify wrapped error identity including `io.ErrShortWrite`.
- Starter contract: native `127.0.0.1:8080`, path-style only, `sigv4_static`, client `local` and separate client/backend env references. Five required variables are documented: `S3PROXY_CLIENT_ACCESS_KEY`, `S3PROXY_CLIENT_SECRET_KEY`, `S3PROXY_TARGET_PRIMARY_ENDPOINT`, `S3PROXY_TARGET_PRIMARY_ACCESS_KEY`, `S3PROXY_TARGET_PRIMARY_SECRET_KEY`. One exact-bucket parser `images`, route `images_rw`, destination `primary`, virtual bucket `images`, and pre-existing backend bucket `images-store` in `us-east-1`; keys are unchanged. Route operations are exactly GetObject, HeadObject, PutObject, DeleteObject, HeadBucket, ListObjectsV2 in that order; client grants additionally include virtual ListBuckets, with only `images_rw` allowed and `images` visible. Dispatch/read preference are `first`, on_match is `stop`.
- Resource contract: explicit 32 MiB per-request and 256 MiB aggregate replay bounds; read-header/read/write/idle deadlines 10s/2m/5m/60s and target complete-stream timeout 2m. Docs explain replay triggers, 413/503 outcomes, memory/concurrency and object-size/link-speed tuning, and distinguish replay bounds from universal upload or total-memory limits.
- Actual workflow: `TestPrintExampleConfigCLI` builds a temporary real executable, invokes printing in an empty working directory with required variables absent, valid synthetic values, and secret-filled malformed values containing quotes, CR/LF/tab, backslashes, Unicode, template/directive syntax and malformed URL escapes. All outputs are byte-identical static HCL with empty stderr and literal env references; the working directory stays empty. The captured output is written by the test harness, then passed unchanged to real `validate --config` and `routes --config` with the documented synthetic client/backend pairs and an ephemeral loopback counting-backend URL. Validation returns exactly `config is valid\n`; topology matches the complete expected single-route JSON schema/order. A listener-address-only fixture variant also passes both commands while that address is occupied. Backend connection count stays zero throughout; no CLI runtime is started.
- Missing prerequisites: each of the five variables is individually omitted from an otherwise valid environment; both real `validate` and `routes` exit 1 with empty stdout, useful value-free validation errors, and no runtime logs. Loaded actual output is checked using production classification, routing, rewrite, authorizer and virtual-bucket listing contracts: all six ordinary operations map correctly (including escaped object keys and root list/head shape), other buckets/routes and undeclared operations are denied, and root discovery is authorized locally with exactly `images` visible and no backend route.
- Documentation: README advertises the command and native workflow; quickstart includes synthetic offline variable setup followed by real credential/backend prerequisites and serve/client instructions, while retaining the distinct manual auth-none walkthrough. Deployment documents `--entrypoint /usr/local/bin/s3proxy`, the required container listener edit to `:8080`, container-reachable endpoints, env-file/nonroot mount considerations, restricted `127.0.0.1:8080:8080` host publishing and external TLS. The image entrypoint is unchanged. Design records the feature as delivered rather than future work.
- Checks passed: `go test -count=1 ./cmd/s3proxy -run TestPrintExampleConfig`; final `go test -count=1 ./cmd/s3proxy ./internal/config ./internal/app` (including existing CLI confidentiality/public-example tests); `make vet test` (all 19 unit packages). Initial focused compilation exposed a test-only operation constant naming error, corrected before passing runs. Scoped `gofmt -l`, tracked-file `git diff --check`, and no-index whitespace checks of both new Go files and this task record passed.
- Preservation/status: all edits used `apply_patch`, prior dirty/untracked work retained, and `CHANGELOG.md` SHA-256 remains `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`. No commits/resets/whole-file restoration, broad formatting, nested agents, live stacks, or installed/dist artifact replacement. Local counting/occupied sockets are test fixtures only; Docker/backend deployment and live object workflows were not run or claimed. No EXAMPLE-01 blocker remains. FINAL-05 remains pending for independent review.

### Task FINAL-05: Independently Verify Validation And Onboarding

Status: completed

Session: fresh independent reviewer; implemented neither predecessor, read the complete task and AGENTS.md, and confirmed LISTENER-01 and EXAMPLE-01 completed before starting. No nested agents. Starting worktree has extensive prior tracked/untracked changes; original CHANGELOG SHA-256 confirmed.

Kind: investigation

Priority: P2 — acceptance/completion gate.

Suggested agent: fresh reviewer who implemented neither preceding task.

Dependencies: LISTENER-01, EXAMPLE-01.

Primary ownership: this task record and bounded acceptance-blocking corrections.

Finding: tightened validation must preserve offline semantics, and the starter must compose with real config/auth/route validation without leaking environment values or misleading container users.

References: tasks above, `AGENTS.md`, prior CLI/confidentiality final reviews.

Requirements:

1. Independently inspect every criterion against code, regression evidence, actual CLI fixtures and docs, including port/service/IPv6 boundaries, safe diagnostics, offline behavior, static output/error propagation, starter permissions and native/container assumptions.
2. Correct bounded acceptance blockers with meaningful evidence; record any findings explicitly. Do not silently expand scope or remove previous work.
3. Run `make vet test`, `GOFLAGS=-count=1 make test-race`, `make build`, `go test -tags integration -run '^$' ./internal/integration/...`, scoped formatting/whitespace and original CHANGELOG hash. Verify the built artifact advertises/emits the new command and its output passes the documented offline workflow using synthetic values.
4. Record a final criterion-by-criterion acceptance matrix and all commands/results/limitations; mark completed only after passing. Coordinator reviews the entire final record afterward.

Acceptance criteria:

- Both tasks delivered with passing acceptance, regression/real-CLI evidence, accurate docs and no hidden blockers.
- Full required checks and built-binary offline workflow pass; live Docker/backend availability is not claimed.
- All three tasks completed with Completion evidence, prior work preserved and CHANGELOG byte-identical.

Verification: requirements above, no Docker/cloud prerequisite.

Review findings (retained even when fixed):

- FINAL-05-F2 (P3, EXAMPLE-01 documentation completeness): `website/docs/operations.md` still listed the four old standard binary entrypoints and omitted the delivered starter command. Added command registration and concise print-to-file/native/Docker prerequisite links, checked against `main.go`, the static template and both onboarding pages. No production behavior change.
- FINAL-05-F1 (P2, bounded LISTENER-01 acceptance blocker; fixed): `listenerPortInRange` saturated at 65535 but kept scanning arbitrarily long numeric prefixes for a later nondigit. Go 1.26.6 `net/port.go:33-49` stops at its 1<<30/uint32 addition-overflow boundaries instead. Consequently `10737418240service` and `4294967296service` passed `validate`/`routes`, then failed `serve` with raw address/runtime output. Regression-first `go test -count=1 ./internal/config ./cmd/s3proxy -run 'TestListenerAddress.*(Invalid|CLI)'` failed in all three new direct-config cases and all 12 new real-CLI combinations (two overflow forms × literal/env × validate/routes/serve). The local numeric parser now follows Go's early overflow boundary without lookup; signed overflow coverage and positive boundary service controls preserve the distinction. Public configuration docs and design explicitly describe it.
- F1 refinement/evidence: further inspection of Go's uint32 arithmetic found that multiplication can wrap before the nondigit (for example `4294967300service`). The first correction using uint64 was too strict for that service form. `go test -count=1 ./internal/config -run 'TestListenerAddressOfflineContract/listener.invalid:4294967300service'` demonstrated the intermediate failure. The final correction follows Go's uint32 service-classification behavior and keeps a separate sticky range flag so a purely numeric `4294967300` still fails the required 0–65535 gate. Positive service controls include `65536service`, `1073741824service`, `4294967295service`, signed prefixes and the wrapping form. Full checks/build/artifact verification were repeated after this refinement and passed. No residual F1 blocker.

### FINAL-05 Acceptance Matrix

All rows were independently inspected against current source, actual test assertions and public docs; predecessor summaries were supporting history, not the acceptance oracle.

| Criterion / requirements                                                                     | Independent source and behavior evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | Result         |
| -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| LISTENER A1; R1, R3: malformed syntax and numeric range fail Load/direct Validate/CLI safely | `internal/config/validate.go:validateListener`, `Load`/`LoadFile`, app `Build`/`ValidateConfig`, and all three CLI entry paths enforce the shared boundary. Fixed diagnostics discard `SplitHostPort` errors. `TestListenerAddressInvalid` exercises missing separator, URL with/without port, brackets, IPv6, extra colons, negative/high/overflow ports, direct Validate and literal/env Load/LoadFile. `TestListenerAddressCLI` now checks 60 literal/env × validate/routes/serve failures, empty stdout, public block/field/cause, no sentinel/parser values or startup logs; artifact additionally checks F1.  | PASS, F1 fixed |
| LISTENER A2; R2: Go-compatible numeric/empty/service/IPv6/zone forms                         | Independently read Go 1.26.6 `net/ipsock.go` and `net/port.go`. Config and real CLI tests preserve 0/1/65535, signs/bare signs/negative zero/leading zeros, wildcard and IPv4, bracketed IPv6 and named/numeric zones, hostnames, empty ports and unresolved services. Config coverage also includes mapped IPv4, bracketed hostnames and underscore hosts. F1 boundary and wrap regressions prevent numeric/service misclassification while all purely numeric out-of-range inputs stay rejected.                                                                                                                  | PASS           |
| LISTENER A2–A3; R2, R4: offline valid occupied/unavailable listeners                         | No lookup, resolver, service lookup, interface check or bind is called in `validateListener`, `Load`, `ValidateConfig`, or `loadTopologyReport`; binding remains in `App.Run`. Actual CLI fixtures accept occupied sockets, `192.0.2.1:8080`, `.invalid` hosts/nonexistent service/zone values, with counting backends remaining at zero connections. `routes_test.go` uses the corrected structurally valid unavailable fixture.                                                                                                                                                                                   | PASS           |
| LISTENER A3; R3: existing confidentiality/public examples preserved                          | Read `diagnostics.go`, native env evaluation in `load.go`, config and CLI confidentiality tests, routes DTO allowlist/tests, and `public_examples_test.go`. Full/focused tests pass; errors preserve public labels/file paths while withholding attribute values. Startup validation precedes app logging.                                                                                                                                                                                                                                                                                                          | PASS           |
| LISTENER A4; R5: docs and regression evidence                                                | Configuration/deployment/design distinguish the earlier syntax gate from runtime resolution, ownership, permissions and port availability. Original regression-first history is retained; reviewer independently reproduced F1 failures before its correction without restoring old work. Public docs now explain the extra overflow/suffix boundary.                                                                                                                                                                                                                                                               | PASS           |
| EXAMPLE A1; R1, R4: deterministic, version-owned, no-prerequisite output                     | `main.go` registers `newPrintExampleConfigCommand`; `example.go` writes one Go constant with newline, with no config/env/app/network/file-management calls. Real subprocess tests use an empty working directory and unset, valid, and quote/control/Unicode/template/malformed-URL secret environments, checking identical bytes, literal env calls, empty stderr and no files. Built artifact repeats these independently.                                                                                                                                                                                        | PASS           |
| EXAMPLE A1; R1: invalid arguments and output failures                                        | `cobra.NoArgs` and command-local flags reject positionals/unknown/config/output flags. Write errors are wrapped and silent short writes become `io.ErrShortWrite`. Tests assert failed/partial/short writer error identity and actual `/dev/full` exit 1; built artifact verifies argument/output exits too.                                                                                                                                                                                                                                                                                                        | PASS           |
| EXAMPLE A2; R2, R5: exact topology and safe missing prerequisites                            | Actual captured HCL passes real validate/routes with documented synthetic variables; JSON exactly matches one ordinal-1 `images_rw` / `images` bucket_exact / `primary` route, six operations in documented order, first/stop/first. Each of five variables omitted separately fails both offline commands safely. Artifact repeats exact quickstart values including `http://127.0.0.1:9000`; zero-backend-connection proof comes from the existing counting fixtures plus inspected offline call paths.                                                                                                           | PASS           |
| EXAMPLE A2; R2, R5: auth, operations, virtual discovery, list/head mapping                   | Read static HCL, config ref normalization/policy validation, authenticator client-map/principal construction and verifier, authorizer, request classification, router, rewrite, handler ListBuckets/route authorization boundaries, and virtual-bucket lister. Generated-output contract tests verify sigv4_static/separate keys, explicit sole-route/bucket grants, Get/Head/Put/DeleteObject + HeadBucket + ListObjectsV2 mapping to `images-store`, unchanged escaped keys, empty list/head key and matching namespace, rejection of other buckets/routes/operations, and local discovery with exactly `images`. | PASS           |
| EXAMPLE A3; R4, R5: no environment disclosure/listener/backend use                           | Actual CLI tests and artifact have identical static output across hostile values; no output includes substituted secrets. `validate`/`routes` use loaded config only; occupied-listener variants and counting-backend assertions pass. No successful serve is needed for this contract.                                                                                                                                                                                                                                                                                                                             | PASS           |
| EXAMPLE A4; R3, R6: native defaults/resources/prerequisites                                  | README, quickstart, configuration/design and printed HCL agree: loopback 127.0.0.1:8080, path-style, five literal env references, separate inbound/upstream keys, us-east-1, pre-created images-store. Explicit 32/256 MiB replay bounds and 10s/2m/5m/60s listener plus 2m complete-target deadlines match app/template/tests. Docs explain replay triggers, 413/503, size/concurrency/link tuning, ordinary upload limitations, shell redirection and separate auth-none walkthrough.                                                                                                                             | PASS           |
| EXAMPLE A4; R6–R7: registration and native/Docker distinction                                | Operations command list corrected (F2); README/quickstart/deployment/design agree with actual registration. Inspected Dockerfile's distroless nonroot image, executable path and serve-specific entrypoint; documented override, network-none offline commands, :8080 container edit, restricted host publication, env-file/mount rules, container-reachable backend and TLS guidance match. Design describes command as delivered.                                                                                                                                                                                 | PASS, F2 fixed |
| FINAL A1; R1–R2: independent review and bounded findings                                     | Fresh reviewer implemented neither predecessor, read complete task + AGENTS, confirmed dependencies completed, marked in_progress before work, and independently inspected all rows. F1/F2 remain explicit above, including intermediate refinement. Only bounded corrections made.                                                                                                                                                                                                                                                                                                                                 | PASS           |
| FINAL A2; R3–R4: required checks and built binary                                            | Final commands/results below pass after the last production edit; artifact advertises/emits command and runs documented synthetic file/validate/routes workflow. Live Docker/backend behavior is explicitly unverified.                                                                                                                                                                                                                                                                                                                                                                                             | PASS           |
| FINAL A3: task statuses, work preservation, CHANGELOG                                        | LISTENER-01 completed; EXAMPLE-01 completed; FINAL-05 completed with this matrix and evidence. Initial/final status inventories retain the extensive existing tracked/untracked work; all edits were apply_patch. CHANGELOG matches the original hash. Coordinator full-record closeout is still the next handoff.                                                                                                                                                                                                                                                                                                  | PASS           |

Completion evidence:

- Reviewer changes: `internal/config/validate.go`, `internal/config/listener_address_test.go`, `cmd/s3proxy/listener_address_test.go`, `website/docs/configuration.md`, `website/docs/operations.md`, `docs/design.md`, and this task file. The required `make build` refreshed only the host `dist/s3proxy` artifact; no release archive/checksum workflow ran. External verification harness: `<repo-root>/tmp/s3proxy-final05-artifact.py`, created via apply_patch; its temporary config files were cleaned by the harness.
- `go test -count=1 ./internal/config ./internal/app ./cmd/s3proxy` — PASS after the initial F1 fix, including public-example/confidentiality/generated-HCL/offline CLI tests. Two intentional failing regression commands and their corrections are recorded under F1 above. The final refinement was covered by the subsequent full uncached race suite.
- `make vet test` — PASS, all 19 unit packages after the final code change.
- `GOFLAGS=-count=1 make test-race` — PASS, all 19 packages uncached after the final code change; no race reports.
- `go test -tags integration -run '^$' ./internal/integration/...` — PASS (`[no tests to run]`, cached); integration-tag compilation only, no live execution.
- `make build` — PASS after the final code change; CGO-disabled, trimpath/buildvcs flags from Makefile, version `079d2aa-dirty`. `./dist/s3proxy --help` advertises `print-example-config`; `./dist/s3proxy version` returns that version.
- `python <repo-root>/tmp/s3proxy-final05-artifact.py` — PASS after final rebuild. Verifies built help; static byte identity with unset/synthetic/malformed environments and no printing-created files; captured print-to-file followed by real `validate --config ./config.hcl` (`config is valid\n`) and exact `routes` JSON; safe failures for each missing required variable; invalid arguments/flags; actual `/dev/full`; safe pre-runtime rejection of both F1 overflow forms in validate/routes/serve. Emitted HCL is 1,388 bytes, SHA-256 `ff04eecfe9c0f72fdcea2ba579a89c562b740e7c3a8ea0c38b18aab11a84555b`.
- Scoped formatting: `gofmt -l` on `internal/config/{validate,listener_address_test}.go` and `cmd/s3proxy/{main,routes,routes_test,listener_address_test,example,example_test}.go` — PASS, empty output. No formatter writes/broad formatting.
- Scoped whitespace: `git diff --check -- internal/config/validate.go cmd/s3proxy/main.go README.md docs/design.md website/docs/configuration.md website/docs/deployment.md website/docs/quickstart.md website/docs/operations.md` — PASS. `git diff --no-index --check /dev/null <file>` on both listener tests, example source/tests, routes source/tests and this task record — no diagnostics. Those untracked-file comparisons exit 1 for file differences; a subprocess driver checked every file individually and required empty stdout/stderr instead of incorrectly short-circuiting an `&&` chain.
- `sha256sum CHANGELOG.md` — unchanged original `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`; final status inventory preserves prior dirty/untracked paths. No resets, commits, restoration, nested agents, Docker/cloud/live-stack actions or release rebuilds.

Limitations and handoff:

- No live Docker, MinIO/SeaweedFS, DNS/service-resolution, interface ownership, successful serving, or real backend credential/bucket availability verification is claimed. Local counting/occupied sockets are test fixtures. Syntactic validation intentionally cannot establish deployment availability.
- Generated auth/access/list/head/discovery behavior is verified at the configuration/production-component contract; existing full unit suites cover HTTP/auth behavior. No new live object lifecycle or container execution was performed.
- Linux/amd64 Go 1.26.6 host build and tests were used; cross-platform release archives and published containers were not rebuilt/audited. The delivered HCL has explicit starter bounds, not universal sizing guarantees.
- No unresolved acceptance blocker or deferred finding remains. F1 and F2 are fixed; all three task statuses are completed. Coordinator must inspect this entire final task record and perform the requested closeout; that handoff is not claimed complete by the reviewer.

## Definition Of Done

All three tasks completed sequentially in fresh sessions with evidence, independent review and coordinator full-record closeout. No unresolved acceptance criterion or CHANGELOG change.

## Coordinator Closeout

Continuation: [Reliable asdf installation and version discovery](20260926-190427-asdf-install-health.md) tracks the next user-requested phase. This phase remains completed.

- Read the entire final task record, including predecessor regressions, FINAL-05's port-parser correction/refinement, documentation correction, acceptance matrix, built-artifact workflow and limitations. Also inspected the final listener validator, static starter command, worktree inventory and CHANGELOG hash. All three tasks are completed with Completion evidence and no unresolved acceptance blocker.
- Confirmed sequential isolated sessions: LISTENER-01 `ses_f1fb4fd68ffeoX09hhwYW9H803`; EXAMPLE-01 `ses_f1f741a54ffeoLY0LPjTeuDWs0`; independent FINAL-05 `ses_f1f6f78d0ffes7tYkn3PocYXF4`. Each session finished before the next started; no nested agents.
- Final required gates passed after the review corrections: `make vet test`, `GOFLAGS=-count=1 make test-race` (19 packages), `make build`, integration-tag compilation, scoped formatting/whitespace and built-binary print/validate/routes checks. Docker deployment and live backend availability remain unverified as documented.
- Coordinator rechecked CHANGELOG SHA-256: `48927af97ba4b4ee87d76c72ce9aa503ef019f91a36c5f3ae50e37ac3bef253e`, unchanged. Previous dirty/untracked paths remain present; no deletion is reported. No commits or live-stack operations occurred.
- Earlier pending/handoff notes are historical and superseded by this closeout. The continuation and requested coordinator final task-file review are complete.
