---
sidebar_position: 8
---

# Deployment

This page covers practical ways to run `s3proxy` in local, containerized, and service-managed environments.

## Deployment Shape

The project is built as:

- a single Go binary named `s3proxy`
- a container image built locally from the repo `Dockerfile`
- versioned container images published as `ghcr.io/egose/s3proxy:<version>` when a release tag is published

The proxy exposes an HTTP S3-compatible endpoint. TLS termination is usually handled by an external load balancer or reverse proxy.

## Local Binary

Build the binary:

```sh
make build
```

Run it:

```sh
./dist/s3proxy serve --config /etc/s3proxy/config.hcl
```

Validate config without starting the server:

```sh
./dist/s3proxy validate --config /etc/s3proxy/config.hcl
```

## Docker

Build the image:

```sh
make docker-build
```

Run it with a mounted config file:

```sh
docker run --rm \
  -p 8080:8080 \
  -v ./config.hcl:/etc/s3proxy/config.hcl:ro \
  -e S3PROXY_CLIENT_CI_ACCESS_KEY=... \
  -e S3PROXY_CLIENT_CI_SECRET_KEY=... \
  -e S3PROXY_TARGET_PRIMARY_ACCESS_KEY=... \
  -e S3PROXY_TARGET_PRIMARY_SECRET_KEY=... \
  s3proxy
```

If your config depends on more `env("...")` values, pass them in as environment variables or via an env file.

## Docker Compose

Example `compose.yaml`:

```yaml
services:
  s3proxy:
    image: s3proxy:latest
    ports:
      - '8080:8080'
    env_file:
      - .env
    volumes:
      - ./config.hcl:/etc/s3proxy/config.hcl:ro
    restart: unless-stopped
```

The example uses the image produced by `make docker-build`; it does not assume a public `latest` image.

Published GHCR images use semantic-version, major/minor, and major tags. The publishing workflow does not create a public `latest` tag. Before publishing, it smoke-tests and scans the exact local image, then verifies the pushed image digest and attaches provenance and SBOM attestations.

## systemd

Example unit file:

```ini
[Unit]
Description=s3proxy
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=s3proxy
Group=s3proxy
WorkingDirectory=/etc/s3proxy
EnvironmentFile=/etc/s3proxy/s3proxy.env
ExecStart=/usr/local/bin/s3proxy serve --config /etc/s3proxy/config.hcl
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Recommended layout:

- binary at `/usr/local/bin/s3proxy`
- config at `/etc/s3proxy/config.hcl`
- env file at `/etc/s3proxy/s3proxy.env`

## Reverse Proxying

`s3proxy` is often run behind an ingress, load balancer, or reverse proxy that handles:

- TLS termination
- public DNS and certificates
- network-level access control
- request logging outside the application process

If you rely on virtual-hosted addressing, make sure the outer layer preserves the bucket-bearing `Host` header shape the proxy expects.

## Restarts And Rollouts

There is no hot config reload in v1.

Use an explicit restart when changing:

- auth clients
- targets
- routes
- virtual buckets
- listener settings

Before rollout:

1. Run `s3proxy validate --config ...`.
2. Confirm every `env("...")` variable is present in the deployment environment.
3. Verify upstream connectivity and backend credentials.
4. Exercise one read path and one write path through the proxy.
5. If using `dispatch = "all"` or `ordered_failover`, test those behaviors before production rollout.

`/healthz` and `/readyz` are unauthenticated endpoints on the main listener. `/readyz` reports only that the process is serving requests; it does not probe target backends. Configure load-balancer health checks against `/readyz`, and restrict access at the network or reverse-proxy layer if needed. The distroless image does not include a shell or HTTP client for an in-container health command.

The process handles `SIGINT` and `SIGTERM` with a 10-second graceful-shutdown window before active connections are forcibly closed. Configure service managers and orchestrators with a termination grace period longer than 10 seconds.

## Production Recommendations

- use `sigv4_static` unless the deployment is fully trusted
- keep client credentials separate from backend target credentials
- mount config files read-only
- set explicit target `timeout` values when using `ordered_failover`, accounting for the fact that the timeout covers the complete upstream response stream
- monitor memory usage for fan-out writes, multi-route writes, concrete SigV4 payload hashes, and unknown-length request bodies
- collect the structured JSON logs written to stdout; no metrics endpoint is exposed in v1
- validate config before every deploy
