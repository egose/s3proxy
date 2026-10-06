# Isolate Sandbox Compose Lifecycle

Created: 2026-09-26 15:10:17 local time

## Objective And Evidence

Ensure s3proxy sandbox startup/teardown cannot remove containers belonging to another repository that also has a `sandbox/` Compose directory. Discovered during the required live verification for SIGN-01 in [S3 Workflow And Codebase Health](20260926-145643-s3-workflow-health.md). This is an independent operational follow-up; no implementation was performed in the SIGN-01 session.

### Task SANDBOX-01: Namespace Every Sandbox Compose Invocation

Status: completed

Kind: defect

Priority: P1 — the documented integration command can remove another project's stopped containers.

Dependencies: none

Primary ownership: `Makefile`, `scripts/run-integration.sh`, sandbox lifecycle tests and operational docs.

Finding: `Makefile` constructs `docker-compose --env-file .env -f ./sandbox/docker-compose.yml` without an explicit project name and defaults startup flags to `up --build --remove-orphans`. `scripts/run-integration.sh` also invokes Compose directly without a project name for init/health container discovery. The derived project name is `sandbox`. During `make sandbox-integration-up` on 2026-09-26, Compose removed pre-existing stopped containers `aiproxy_postgres`, `aiproxy_keycloak`, and `aiproxy_keycloak_provision` as orphans. The two containers running before the command (`aiproxy-review-20260926-145524` and `orchestration`) were still running afterward. The removed containers were not recreated: their original configuration was not inspected. The command did not request volume deletion.

References: `Makefile` (`COMPOSE`, `UP_FLAGS`, sandbox lifecycle targets); `scripts/run-integration.sh` (direct `docker-compose` calls, cleanup trap); `sandbox/docker-compose.yml` (no explicit project name).

Requirements:

1. Use one explicit s3proxy-specific Compose project identity consistently for startup, discovery, logs, teardown, and integration cleanup. Keep direct script invocations and Make targets aligned; respect any supported override consistently.
2. Prevent orphan removal from affecting the historical shared `sandbox` project. Document migration of existing containers without deleting unfamiliar containers or volumes automatically.
3. Preserve MinIO, SeaweedFS, Azurite, fake-gcs-server, and s3-error services, existing credential handling, and integration exit-code propagation.
4. Add a regression with an isolated fake Compose command or a disposable, uniquely named Docker project proving that lifecycle/discovery invocations select the intended project and leave unrelated containers untouched. Do not reproduce against the user's shared `sandbox` project.

Acceptance criteria:

- Every sandbox Compose invocation selects the same explicit project identity.
- An unrelated container cannot be removed by the s3proxy startup/cleanup flow.
- Failed integration runs still clean up their own stack and propagate the failure.
- Migration/operational docs explain the historical collision and safe handling of existing containers.

Verification: run from `<repo-root>`; `bash -n scripts/run-integration.sh`, `shellcheck scripts/run-integration.sh`, the new lifecycle regression, `git diff --check`, then `make sandbox-integration-up` only after project isolation is established and existing container-name conflicts are handled without deleting unrelated work.

Execution record:

- Fresh sequential SANDBOX-01 session started; read this task, the main plan, `AGENTS.md`, and the user task-as-you-go skill. Existing REWRITE-01/SIGN-01 worktree changes are preserved. Live lifecycle commands are gated on the isolation fix and fake regression.

Completion evidence:

- Changed implementation: `scripts/sandbox-compose.sh` (new shared wrapper), `Makefile`, `scripts/run-integration.sh`, `sandbox/docker-compose.yml`, `.github/workflows/test.yml`. Changed regression/docs: `tests/sandbox-lifecycle.sh` (new), `README.md`, `AGENTS.md`, `website/docs/operations.md`, this task and the main plan's linked execution amendment/dependencies. Prior REWRITE-01/SIGN-01 changes were preserved; no commits or CHANGELOG edits were made.
- Identity resolution: every repository sandbox Compose call, including CI config validation, selects explicit `s3proxy-sandbox` through the wrapper. Plugin `docker compose` is preferred, standalone `docker-compose` is the fallback, and `SANDBOX_COMPOSE` accepts one executable path. `SANDBOX_PROJECT_NAME` is supported through environment and Make assignments, restricted to `s3proxy-[a-z0-9][a-z0-9_-]*`; ambient/`.env` `COMPOSE_PROJECT_NAME` cannot redirect it. Integration startup, discovery, and cleanup pin the same override values before sourcing credentials. Init discovery now includes stopped one-shot containers (`ps -a -q`).
- In-scope lifecycle corrections found during implementation: removed automatic `--remove-orphans` from startup/destroy and the unscoped `docker image prune -f` from destroy/reset. Removed five fixed container names so Compose generates project-scoped names without legacy name collisions; all seven services (including both init jobs), ports, credentials, and volume declarations remain. Integration cleanup kills/waits for its own child PID rather than trusting a potentially pre-existing PID file.
- Fake regression: `make test-sandbox` — PASS. It runs in a temporary copied workspace with fake Docker/Compose/Go/curl, never the real Docker daemon. Three frontends (plugin, standalone fallback, explicit executable), two identities (default and override), every lifecycle target, six direct-runner outcomes (success, test/startup/init/build failure, and test failure plus cleanup failure), Make integration failure propagation, `.env` override resistance, credential loading, proxy termination, and invalid project rejection are covered. Unrelated markers in both historical `sandbox` and another project, plus a selected-project orphan, survive. Direct test failure returns 37; Make returns 2 and reports Error 37; cleanup's simulated 29 cannot replace the original test failure.
- Sensitivity evidence: temporarily removing only the wrapper's explicit `--project-name` made `make test-sandbox` fail (Make exit 2/Error 1); restoring it passed. This fault injection stayed entirely in the fake regression. Initial shellcheck found SC2251 in a test assertion; it was corrected to an explicit conditional and all final lint checks passed.
- Required/local checks from `<repo-root>`: `bash -n scripts/run-integration.sh`, `bash -n scripts/sandbox-compose.sh`, and `bash -n tests/sandbox-lifecycle.sh` — PASS; `shellcheck scripts/run-integration.sh scripts/sandbox-compose.sh tests/sandbox-lifecycle.sh` — PASS; `make test-sandbox` — PASS; `make vet test` — PASS (all 16 unit packages); `make test-race` — PASS (all 16 packages); `make build` — PASS; `go test -tags integration -run '^$' ./internal/integration` — PASS; `actionlint` — PASS; `make check-toolchain` — PASS; `shellcheck bin/* scripts/*.sh tests/*.sh .bin/*.sh` — PASS; `git diff --check` — PASS; `git diff --exit-code -- CHANGELOG.md` — PASS, empty diff.
- Compose compatibility checks: `docker info --format '{{.ServerVersion}}'` — 29.7.2; `docker compose version` — v5.4.0; `docker-compose version` — v5.3.1; `bash scripts/sandbox-compose.sh config --quiet` and `SANDBOX_COMPOSE=docker-compose bash scripts/sandbox-compose.sh config --quiet` — both PASS. Legacy standalone CLI selection is fake-tested; a historical Python Compose v1 installation was not available for a live check.
- Safe live verification, only after restored isolation and the fake regression passed: an in-memory Python/subprocess preflight inspected the wrapper's JSON config, all Docker container IDs/labels/names/running states, volume/network inventories, and bound/released required host ports. It confirmed all seven services, no fixed container names, no existing selected-project containers/volumes, and free ports 8080, 8082, 8333, 9000, 9001, 9333, 10000, 10001, 18081. No legacy resource needed removal or migration.
- `make sandbox-integration-up` — PASS, exit 0, all 28 top-level integration tests including eight escaped-key MinIO/SeaweedFS round trips. Startup and teardown visibly used only `s3proxy-sandbox-*` containers and `s3proxy-sandbox_s3proxy` network. Before/after assertions passed: all 26 pre-existing container IDs/names/running states, all 288 pre-existing volumes, and all seven pre-existing networks were preserved; no integration containers remained. Four new project-named volumes and one new anonymous image volume were retained by normal `down`; no volume removal was requested.
- Final proxy teardown check: `test ! -e dist/s3proxy-integration.pid` — PASS; a Python socket bind/release on `127.0.0.1:8082` — PASS, confirming the proxy listener was gone.
- Operational docs explain historical collisions, project overrides, generated names, ownership/port inspection, fresh versus retained legacy volumes, and owner-led migration without shared-project cleanup. Limitations: fixed host ports and `dist/` PID/log paths still require sequential local stacks; project labels cannot distinguish two callers intentionally reusing the same identity. The previously removed `aiproxy_postgres`, `aiproxy_keycloak`, and `aiproxy_keycloak_provision` were not recreated without original configuration. Their historical incident remains recorded in SIGN-01. LIST-01 may now proceed; final review must include this task's evidence.

Independent FINAL-01 audit (2026-09-26):

- Inspected actual Make/runner/wrapper/CI invocations, service-name changes, fake lifecycle assertions and migration docs against all four acceptance criteria. Re-ran shell syntax/shellcheck, `make test-sandbox`, both installed Compose frontends' `config --quiet`, `actionlint` and toolchain checks — PASS. SANDBOX-01 remains completed.
- Independently ran `SANDBOX_PROJECT_NAME=s3proxy-final-20260926-154814 make sandbox-integration-up` through an in-memory preflight/postcheck wrapper. Fresh project ownership, all seven services/no fixed names and all required free ports were asserted first. All 29 top-level live integration tests passed; canonical cleanup removed its containers/network and released proxy port/PID. All 28 pre-existing container IDs/names/running states, 297 volumes and eight networks survived unchanged. New test volumes were retained. No unfamiliar or shared-project resources were modified.
- Exact broader commands/results and remaining phase blockers are recorded under FINAL-01 in [the main plan](20260926-145643-s3-workflow-health.md). Newly confirmed redirect/diagnostic findings are separate from sandbox isolation; the prior legacy-container restoration and fixed-port limitations remain explicit.

Resumed FINAL-01 confirmation after both review follow-ups (2026-09-26):

- SANDBOX-01 remains completed. Shell syntax/shellcheck and `make test-sandbox` passed again. The final live run used fresh identity `s3proxy-final-20260926-162604` after ownership/port preflight: `make sandbox-integration-up` passed all 29 top-level tests and canonical teardown. All 27 containers (IDs/names/running states), 301 volumes and eight networks present in this run's baseline were preserved. Its containers/network and proxy port/PID were cleaned up; new volumes were retained.
- Both review follow-ups and FINAL-01 are now completed; earlier phase-blocker notes above are historical. See [the resumed FINAL-01 completion evidence](20260926-145643-s3-workflow-health.md#resumed-final-01-completion-evidence) and [follow-up Definition of Done confirmation](20260926-155056-final-review-follow-ups.md#definition-of-done). Historical restoration/fixed-port limitations are retained; no unfamiliar resources were changed.
