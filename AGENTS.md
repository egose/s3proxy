# AGENTS.md

Operational guide for AI agents (and humans) working in this repo.

## Build & run

| Command                                      | Effect                                                    |
| -------------------------------------------- | --------------------------------------------------------- |
| `make build`                                 | Build `dist/s3proxy` for the host platform (CGO disabled) |
| `pnpm release:build -- --version <version>`  | Build archives and `SHA256SUMS` for all release targets   |
| `pnpm release:verify -- --version <version>` | Verify release archives (`--reproducibility` rebuilds)    |
| `make build-all VERSION=<version>`           | Compatibility wrapper for `pnpm release:build`            |
| `make run CONFIG=path/to/config.hcl`         | `go run` the server against a config                      |
| `make validate CONFIG=path/to/config.hcl`    | Load + validate config without serving                    |
| `make docker-build`                          | Multi-stage container build as `s3proxy:$(VERSION)`       |
| `make docker-run CONFIG=path/to/config.hcl`  | Run the container image with a mounted config             |

The HCL config exposes `env("VAR")` as a native function returning a literal
string during evaluation; environment contents are never parsed as HCL source
or templates. Unset variables return an empty string, then ordinary field
validation applies. The loader parses original source bytes to preserve
diagnostic locations. Run `set -a; . ./.env; set +a` before invoking the binary
locally so env vars resolve.

## Lint / typecheck / test

| Command                      | Effect                                                                       |
| ---------------------------- | ---------------------------------------------------------------------------- |
| `make vet`                   | `go vet ./...`                                                               |
| `make test`                  | `go test ./...` (unit tests only)                                            |
| `make test-race`             | `go test -race ./...`                                                        |
| `make cover`                 | Unit tests with coverage profile at `dist/coverage.out`                      |
| `make test-integration`      | `go test -tags integration ./internal/integration/...` (requires live stack) |
| `make test-integration-race` | Same with `-race` (use to verify the fan-out path)                           |

`make vet test` is the default pre-commit sanity check; run it after any
non-trivial change. There is no separate typecheck target — Go's compiler
is the typecheck, and `make build` exercises it.

Release builds are configured in `go-release.json`. Keep `dist/SHA256SUMS`
archive-only until `pnpm release:verify` completes; the release workflow adds
the SBOM checksum afterward for publication.

## Integration tests

The integration suite is build-tagged `integration` so it is skipped by
`make test`. It exercises the proxy end-to-end against the sandbox
docker-compose stack (MinIO + SeaweedFS).

```sh
cp .env.example .env
make sandbox-integration-up     # one-shot: stack + proxy + tests + teardown
# OR iterative:
make sandbox-up DAEMON=true
make build
make test-integration           # repeat as needed
make sandbox-down
```

`make sandbox-integration-up` is the canonical CI-shaped entry point. It
sources `.env` into the proxy's environment, launches s3proxy as a child
of the integration runner,
runs `go test -tags integration`, then tears down the proxy AND the
docker-compose stack, propagating the test's exit code.

## Sandbox

| Command                         | Effect                                        |
| ------------------------------- | --------------------------------------------- |
| `make sandbox-up [DAEMON=true]` | `docker compose up` the sandbox stack         |
| `make sandbox-down`             | Stop containers, keep volumes                 |
| `make sandbox-destroy`          | Stop and remove containers + volumes + images |
| `make sandbox-reset`            | Destroy then up                               |
| `make sandbox-logs [tail=N]`    | Recent sandbox logs                           |
| `make sandbox-logs-follow`      | Tail sandbox logs                             |
| `make sandbox-ps`               | List running sandbox services                 |

The sandbox stack lives in `sandbox/docker-compose.yml`. Keep existing
services (azurite, fake-gcs-server) even when adding new ones — they are
referenced by other tests/projects in this environment.

Every sandbox Compose call goes through `scripts/sandbox-compose.sh` with
explicit project `s3proxy-sandbox`. It prefers `docker compose`, with a
`docker-compose` fallback; `SANDBOX_COMPOSE` may specify one executable path.
Use `SANDBOX_PROJECT_NAME=s3proxy-<suffix>` in the environment or Make command
line for an alternate identity, consistently across lifecycle commands.
The wrapper ignores `COMPOSE_PROJECT_NAME` and rejects identities outside the
`s3proxy-` namespace. The integration runner pins its selection before loading
credentials from `.env`. Run `make test-sandbox` for the fake lifecycle regression.

Legacy stacks used the shared `sandbox` project. Never run shared-project
orphan cleanup or prune resources to migrate them. Inspect ownership and ports;
leave unfamiliar containers/volumes intact. Migration steps and fixed-port
limitations are in `website/docs/operations.md#sandbox-isolation-and-migration`.

## Conventions

- **No comments** in source files unless the surrounding code dictates
  otherwise — the design doc at `docs/design.md` holds the rationale.
- Module path: `github.com/egose/s3proxy`.
- All HCL blocks use two-label syntax: `listener "http" "public" {}`.
- Inbound and outbound SigV4 use the AWS SDK v2 signer with
  `DisableURIPathEscaping = true` for S3 canonical paths, including presigning.
- Outbound SigV4 uses `UNSIGNED-PAYLOAD`; the `Content-Length` header MUST be set explicitly
  before signing or S3 backends reject with `SignatureDoesNotMatch`.
- Inbound SigV4 verification re-signs with the stored secret key using the
  request's `X-Amz-Date` and compares `Authorization` headers; only the
  `SignedHeaders` from the original auth header are included.
- The dispatcher buffers write bodies into memory so multiple destinations
  can be replayed; reads are never fan-out.
- Multi-destination write: `dispatch = "all"`. Any destination failure
  surfaces the upstream's Primary response body to the client (not a
  generic 502).
- `path_prefix` matches `RawPath == prefix` OR `RawPath` starts with
  `prefix + "/"`. Strict — `/replica` does NOT match `/replicate/...`.
