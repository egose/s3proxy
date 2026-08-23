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

`serve` and `validate` also accept `-c` as shorthand for `--config`. These commands do not accept positional arguments.

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

The one-shot target tears down the proxy and sandbox after the tests. If setup fails before the tests begin, run `make sandbox-integration-down` to clean up any resources that were already started.

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

## Runtime Behavior

Important operational behaviors:

- only one listener is supported
- config changes require a restart; there is no hot reload in v1
- request bodies that need replay are buffered in memory up to `listener.replay_body_max_bytes` per request and `listener.replay_body_aggregate_max_bytes` across the process
- reads use one effective backend even when a route has multiple destinations
- target `timeout` is a deadline for the complete upstream exchange, including streaming the response body, and also affects failover timing for `ordered_failover`

Replay buffering is used for fan-out writes, writes matched by multiple routes, inbound SigV4 requests with a concrete payload hash, and outbound requests whose body length is unknown. If the per-request replay limit is exceeded, the proxy returns `413 EntityTooLarge` instead of attempting the upstream request. If the process aggregate replay budget is exhausted, the proxy returns `503 SlowDown` immediately instead of blocking request goroutines.

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
4. Start the proxy and confirm `ListBuckets` and one object read/write path.
5. If using replication or failover, run the integration suite against the sandbox.
