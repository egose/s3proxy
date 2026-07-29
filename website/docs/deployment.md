---
sidebar_position: 8
---

# Deployment

This page covers practical ways to run `s3proxy` in local, containerized, and service-managed environments.

## Deployment Shape

The project ships as:

- a single Go binary named `s3proxy`
- a container image built from the repo `Dockerfile`

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

## Production Recommendations

- use `sigv4_static` unless the deployment is fully trusted
- keep client credentials separate from backend target credentials
- mount config files read-only
- set explicit target `timeout` values when using `ordered_failover`
- monitor memory usage if you fan out large writes
- validate config before every deploy
