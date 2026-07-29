---
sidebar_position: 7
---

# Operations

This page covers the commands and runtime behavior that matter most when working on or operating `s3proxy`.

## Build And Run

Common commands:

```sh
make build
make build-all
make run CONFIG=path/to/config.hcl
make validate CONFIG=path/to/config.hcl
```

The standard binary entrypoints are:

```sh
s3proxy serve --config /etc/s3proxy/config.hcl
s3proxy validate --config /etc/s3proxy/config.hcl
s3proxy version
```

`make build` produces `dist/s3proxy` for the host platform with CGO disabled.

## Environment Variables

The config loader inlines `env("VAR")` before HCL parsing.

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
make cover
```

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

Iterative flow:

```sh
make sandbox-up DAEMON=true
make build
make test-integration
make test-integration-race
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

## Runtime Behavior

Important operational behaviors:

- only one listener is supported
- config changes require a restart; there is no hot reload in v1
- request bodies that need replay are buffered in memory up to `listener.replay_body_max_bytes`
- reads use one effective backend even when a route has multiple destinations
- target `timeout` directly affects failover timing for `ordered_failover`

If the replay limit is exceeded, the proxy returns `413 EntityTooLarge` instead of attempting the upstream request.

## Logging And Diagnostics

Use `s3proxy validate --config ...` before rollouts to catch configuration errors such as:

- invalid parser definitions
- unknown target or route references
- unsupported operation names
- invalid auth mode or missing clients

For integration troubleshooting, `make sandbox-logs` and `make sandbox-logs-follow` are the fastest way to inspect backend behavior.

## Suggested Local Checklist

1. Load environment variables used by `env("...")`.
2. Run `make vet test`.
3. Run `s3proxy validate --config ...`.
4. Start the proxy and confirm `ListBuckets` and one object read/write path.
5. If using replication or failover, run the integration suite against the sandbox.
