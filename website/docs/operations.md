---
sidebar_position: 7
---

# Operations

This page covers the commands and runtime behavior that matter most when working on or operating `s3proxy`.

## Build And Run

Common commands:

```sh
make build
pnpm release:build -- --version dev
pnpm release:verify -- --version dev --reproducibility
make run CONFIG=path/to/config.hcl
make validate CONFIG=path/to/config.hcl
```

The standard binary entrypoints are:

```sh
s3proxy serve --config /etc/s3proxy/config.hcl
s3proxy validate --config /etc/s3proxy/config.hcl
s3proxy routes --config /etc/s3proxy/config.hcl
s3proxy print-example-config
s3proxy version
```

`serve`, `validate`, and `routes` also accept `-c` as shorthand for `--config`. These commands require a config path and do not accept positional arguments.

`print-example-config` takes no arguments and prints a static authenticated starter
with literal environment references, without loading config or starting the runtime.
Use `s3proxy print-example-config > config.hcl`, then follow the
[native starter workflow](quickstart.md#authenticated-binary-only-starter) for the
five required variables, pre-existing backend bucket, and offline validation.
For containers, follow the [Docker starter instructions](deployment.md#authenticated-starter-in-docker)
for the executable entrypoint override and the required listener change to `:8080`.

`make build` produces `dist/s3proxy` for the host platform with CGO disabled.

`go-release.json` defines the release target matrix and archive contract.
`pnpm release:build` creates deterministic `s3proxy-<os>-<arch>.tar.gz`
archives and `SHA256SUMS`; `pnpm release:verify` checks those artifacts and
their exact version output. Add `--reproducibility` to perform two independent
rebuilds and compare archive hashes. `make build-all VERSION=<version>` remains
a compatibility wrapper for the package build.

Keep `SHA256SUMS` archive-only through package verification. The release
workflow appends the generated SBOM checksum only after verification and then
publishes the archives, checksum manifest, and SBOM together.

## Offline Route Topology

Before a rollout, inspect ordered routing and replication with:

```sh
s3proxy routes --config /etc/s3proxy/config.hcl
# Equivalent shorthand; suitable for saving or comparing topology:
s3proxy routes -c /etc/s3proxy/config.hcl > routes.json
```

The command fully loads and validates the configuration, including credentials,
auth policies, bucket references, and rewrites. Load any environment variables
first, just as for `validate`. It prints only JSON on stdout, with no runtime logs,
app construction, listener binding, or backend calls. Invalid config returns a
nonzero exit status with existing safe diagnostics on stderr and no report on
stdout. File/argument errors and output-write failures also return nonzero; an
output failure can leave a partial file, so check the exit status before using it.

The schema is an object with one `routes` array. Each route contains:

| Field             | Meaning                                                                                                               |
| ----------------- | --------------------------------------------------------------------------------------------------------------------- |
| `ordinal`         | One-based position in config declaration order.                                                                       |
| `label`           | Source-literal route declaration label.                                                                               |
| `parser`          | Object with resolved declaration `label` and `kind`: `path_prefix`, `bucket_exact`, `bucket_regex`, or `host_suffix`. |
| `operations`      | Exact case-sensitive operation names, in configured order.                                                            |
| `destinations`    | Resolved target declaration labels, in configured order.                                                              |
| `dispatch`        | Required `first` or `all`.                                                                                            |
| `on_match`        | Required `stop` or `continue`.                                                                                        |
| `read_preference` | `first`, `random`, `hash`, or `ordered_failover`; omitted/empty config values resolve to `first`.                     |

For example, a route declared as `all`, using parser `path_prefix` named `all`,
with `operations = ["GetObject"]`, destination `primary`, `dispatch = "first"`,
`on_match = "stop"`, and no read preference prints:

```json
{
  "routes": [
    {
      "ordinal": 1,
      "label": "all",
      "parser": {
        "label": "all",
        "kind": "path_prefix"
      },
      "operations": ["GetObject"],
      "destinations": ["primary"],
      "dispatch": "first",
      "on_match": "stop",
      "read_preference": "first"
    }
  ]
}
```

JSON uses two-space indentation and a trailing newline. Repeated invocations
produce identical bytes for the same evaluated topology. A valid config with no
routes prints `{"routes": []}` (formatted across lines). Route, operation, and
destination arrays are never sorted; order matters. Bare and qualified references
resolve to the same declaration labels, including references computed from `env()`.

Operation filters never mean “all operations”: missing/empty lists, `*`, duplicates,
and unsupported names fail validation. Valid operation names are `GetObject`,
`HeadObject`, `PutObject`, `DeleteObject`, `HeadBucket`, `ListObjectsV2`, and
`ListBuckets`. These are configured filters, not a claim that each request reaches
that route: `ListBuckets` is handled locally, earlier matches can stop evaluation,
and authorization can deny requests. Writes may fan out with `dispatch = "all"`;
reads still select one backend. The report includes the configured read preference,
not a simulated effective destination.

Only declaration labels and validated finite-vocabulary fields (plus ordinal) are
exposed. Credentials, endpoints, regions, listener addresses, parser match strings
and regexes, rewrite rules/templates, visible bucket values, environment names and
arbitrary values, and raw reference expressions are excluded. A qualified reference
such as an environment-derived prefix followed by `.primary` prints only the
resolved target declaration label `primary`. Finite enum/operation values may be
shown even when supplied by the environment. Keep secrets out of declaration
labels and config filenames: labels are public report metadata, and filenames and
labels are public diagnostic metadata.

This is configured topology, not request-routing simulation or a backend health
or authorization check. It intentionally cannot explain regex captures or rewrite
results, and it does not prove credentials work against a backend.

## Environment Variables

The config loader evaluates `env("VAR")` as a native HCL function returning a
literal string. Environment contents are never parsed as HCL source or
templates. Unset variables return an empty string and are then subject to
ordinary field validation. Parse/decode diagnostics use original file locations.

If you run locally with a `.env` file, load it before invoking the proxy:

```sh
set -a; . ./.env; set +a
```

## Docker

Build and run with the repo helpers:

```sh
make docker-build
make docker-run CONFIG=path/to/config.hcl
```

The image mounts the config file and runs the same CLI entrypoint as the local binary.

## Unit Test And Validation Commands

```sh
make vet
make test
make test-race
mkdir -p dist
make cover
```

`make cover` writes `dist/coverage.out`, so `dist/` must already exist. `make build` also creates it.

The standard local sanity check is:

```sh
make vet test
```

There is no separate typecheck target. A successful Go build is the typecheck.

## Integration Tests

The integration suite is build-tagged `integration` and is skipped by `make test`.

Canonical one-shot flow:

```sh
cp .env.example .env
make sandbox-integration-up
```

The one-shot target tears down its proxy and sandbox after success or failure, including partial startup failures. The runner preserves the test/setup exit status; Make reports a failed recipe as exit 2 and prints the runner's status. `make sandbox-integration-down` is also available for manual cleanup after an interrupted session.

Iterative flow:

```sh
make sandbox-up DAEMON=true
make build
set -a; . ./.env; set +a
./dist/s3proxy serve --config sandbox/integration-config.hcl
```

Keep the proxy running and use another terminal for repeated test runs:

```sh
set -a; . ./.env; set +a
make test-integration
make test-integration-race
```

When finished, stop the proxy and tear down the sandbox:

```sh
make sandbox-down
```

The sandbox stack exercises the proxy end to end against MinIO and SeaweedFS.

## Sandbox Commands

Useful sandbox helpers:

```sh
make sandbox-up
make sandbox-down
make sandbox-destroy
make sandbox-reset
make sandbox-logs
make sandbox-logs-follow
make sandbox-ps
```

The sandbox compose file lives at `sandbox/docker-compose.yml`.

### Sandbox Isolation And Migration

All Make helpers, integration discovery/cleanup, and CI validation use
`scripts/sandbox-compose.sh`. It passes `--project-name s3proxy-sandbox` explicitly,
prefers the `docker compose` plugin, and falls back to standalone `docker-compose`.
For direct Compose operations, use the same wrapper, for example:

```sh
bash scripts/sandbox-compose.sh config --quiet
bash scripts/sandbox-compose.sh ps -a
```

An alternate project must match `s3proxy-[a-z0-9][a-z0-9_-]*`. Set it in the shell
environment or on every Make command line; use a name dedicated to this checkout:

```sh
export SANDBOX_PROJECT_NAME=s3proxy-my-worktree
make sandbox-up DAEMON=true
make sandbox-ps
make sandbox-down
```

`SANDBOX_COMPOSE=/path/to/docker-compose` optionally selects a single executable
(no embedded arguments). Both overrides apply to direct script calls and Make
targets. Set them before invoking the runner, rather than inside `.env`.
`COMPOSE_PROJECT_NAME`, including a value loaded from `.env`, cannot replace the
explicit project. Integration cleanup retains the selection used at startup,
even after `.env` is sourced for proxy/test credentials.

Containers now use Compose-generated project/service names instead of the old
fixed `s3proxy_*` container names. Use `ps`/`logs` through the wrapper to discover
them. Networks and new named volumes are also project-scoped. Fixed host ports
(including the proxy's 8082) still permit only one stack per host at a time;
a different project name does not isolate host ports or the local `dist/` PID/log
files. MinIO, SeaweedFS, Azurite, fake-gcs-server, and s3-error remain included.

Historically the directory-derived identity was `sandbox`, shared with other
repositories. Startup's `--remove-orphans` could delete their stopped containers.
The helpers now omit orphan removal, and destroy/reset no longer perform global
image pruning. `sandbox-down` retains project volumes; `sandbox-destroy` and
`sandbox-reset` explicitly remove the selected stack's volumes and service images.

For an existing installation:

1. Inspect existing resources without changing them:
   `docker ps -a --filter label=com.docker.compose.project=sandbox` and
   `docker volume ls --filter label=com.docker.compose.project=sandbox`.
   Inspect individual container labels (`com.docker.compose.service`,
   `com.docker.compose.project.config_files`, and
   `com.docker.compose.project.working_dir`), mounts, and published ports to
   establish ownership. A shared project label or familiar name alone is not proof.
2. Leave unfamiliar containers, volumes, and services intact. Do not run `down`,
   `--remove-orphans`, or volume/system prune against the shared `sandbox` project.
   If a legacy container occupies a required port, resolve it with its owner;
   stop/remove only individually verified, owned container IDs after preserving
   any needed data. A name/port conflict is not permission to remove a container.
3. Start the isolated project once its host ports are available. It creates fresh
   volumes such as `s3proxy-sandbox_s3proxy_minio`; old `sandbox_s3proxy_*` volumes
   remain intact and are not adopted automatically. Back up and explicitly migrate
   needed data with the owner's original configuration. Sandbox initialization
   recreates test buckets, so it is not a data-preserving migration tool.
4. Containers already removed by a historical orphan cleanup cannot be restored
   from this repository alone. Obtain their original configuration from their
   owner before attempting recreation; retained volumes do not supply that config.

`make test-sandbox` exercises all lifecycle targets and integration cleanup using
fake Docker/Compose commands in a temporary workspace. It requires no Docker
daemon and checks plugin/standalone selection, project overrides, unrelated
resource preservation, failure status propagation, and proxy cleanup.

## Runtime Behavior

Important operational behaviors:

- only one listener is supported
- config changes require a restart; there is no hot reload in v1
- request bodies that need replay are buffered in memory up to `listener.replay_body_max_bytes` per request and `listener.replay_body_aggregate_max_bytes` across the process
- reads use one effective backend even when a route has multiple destinations
- target `timeout` is a deadline for the complete upstream exchange, including streaming the response body, and also affects failover timing for `ordered_failover`

Replay buffering is used for fan-out writes, writes matched by multiple routes, inbound SigV4 requests with a concrete payload hash, and outbound requests whose body length is unknown. If the per-request replay limit is exceeded, the proxy returns `413 EntityTooLarge` instead of attempting the upstream request. If the process aggregate replay budget is exhausted, the proxy returns `503 SlowDown` immediately instead of blocking request goroutines.

For memory sizing, the aggregate replay budget accounts for payload bytes, not
process RSS. Unknown-length uploads coalesce fragmented reads into 32 KiB chunks;
retained payload capacity equals the buffered byte count, including an exact-sized
final chunk. Allow exactly one additional transient 32,768-byte scratch buffer per
active unknown-length buffering operation, plus chunk metadata, request bookkeeping,
and Go allocator overhead. There is no extra retained final-chunk capacity allowance.
Metadata scales with payload chunk count, not network read count. Unknown-length
sources returning no bytes and no error 100 times consecutively fail instead of
spinning; partial reservations are released on failure or cancellation.

## Logging And Diagnostics

Use `s3proxy validate --config ...` before rollouts to catch configuration errors such as:

- invalid parser definitions
- unknown target or route references
- unsupported operation names
- invalid auth mode or missing clients

For integration troubleshooting, `make sandbox-logs` and `make sandbox-logs-follow` are the fastest way to inspect backend behavior.

The proxy writes structured JSON logs to stdout. Each request receives an `X-Request-Id` response header; an inbound `X-Request-Id` is preserved, otherwise the proxy generates one. Request completion records include the method, status, response bytes, and duration. Dispatch records include route, operation, target, status, and sanitized errors when applicable. There is no runtime setting for log level, format, or destination in v1.

The proxy does not expose a Prometheus or other metrics endpoint in v1. Collect stdout logs and process or container metrics externally.

## Suggested Local Checklist

1. Load environment variables used by `env("...")`.
2. Run `make vet test`.
3. Run `s3proxy validate --config ...`.
4. Run `s3proxy routes --config ...` and review route/destination order and modes.
5. Start the proxy and confirm `ListBuckets` and one object read/write path.
6. If using replication or failover, run the integration suite against the sandbox.
