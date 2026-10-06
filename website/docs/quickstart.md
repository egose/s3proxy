---
sidebar_position: 2
---

# Quickstart

Choose the authenticated starter for binary-only onboarding, or the separate
manual auth-none walkthrough below for trusted local testing.

## Authenticated Binary-Only Starter

With `s3proxy` installed on your PATH (or substitute `./dist/s3proxy` after
`make build`), print the version-owned HCL:

```sh
s3proxy print-example-config > config.hcl
```

Only your shell writes the file. The no-argument command prints deterministic HCL
with a trailing newline; it never loads configuration, evaluates environment
values, generates credentials, binds a listener, or contacts a backend. Printing
works with every variable unset. Unsupported arguments/flags and output failures
return nonzero; a failed output device may have accepted partial output.

### Set Variables And Inspect Offline

The following **synthetic values are for offline inspection only**. They do not
provision a backend or usable deployment credentials:

```sh
export S3PROXY_CLIENT_ACCESS_KEY=synthetic-client-access
export S3PROXY_CLIENT_SECRET_KEY=synthetic-client-secret
export S3PROXY_TARGET_PRIMARY_ENDPOINT=http://127.0.0.1:9000
export S3PROXY_TARGET_PRIMARY_ACCESS_KEY=synthetic-backend-access
export S3PROXY_TARGET_PRIMARY_SECRET_KEY=synthetic-backend-secret

s3proxy validate --config ./config.hcl
s3proxy routes --config ./config.hcl
```

All five variables are required when loading the printed HCL. Missing variables
fail validation safely; values stay literal even if they contain quotes or HCL
template syntax. `validate` prints `config is valid`. Neither offline command
requires a running backend or an available listener port. `routes` reports exactly
one route:

- ordinal `1`, label `images_rw`, parser `images` of kind `bucket_exact`
- operations, in order: `GetObject`, `HeadObject`, `PutObject`, `DeleteObject`,
  `HeadBucket`, `ListObjectsV2`
- destination `primary`, dispatch `first`, on_match `stop`, read_preference `first`

### Prepare The Backend And Serve Natively

Before serving, replace the synthetic client key pair with your own private pair
and replace the backend endpoint/key pair with credentials for your backend.
Keep the two pairs separate: clients sign with `S3PROXY_CLIENT_*`, while the proxy
signs upstream requests with `S3PROXY_TARGET_PRIMARY_*`. The endpoint is the backend
service URL, not a bucket URL. Use HTTPS for a remote backend.

Provision **`images-store` on the backend in advance**, with permission for the
backend key to read/write/delete objects and list/head that bucket. The starter
uses region **`us-east-1`** and path-style addressing on both sides; edit the target
region if your backend requires another. The proxy does not create buckets.

The native listener is **`127.0.0.1:8080`**. Only bucket `images` matches the single
route; its keys map unchanged into `images-store`. The sole client `local` has an
explicit route grant, visible bucket `images`, the six routed operations above,
and `ListBuckets` permission. ListBuckets is served locally from virtual bucket
metadata, not forwarded through a catch-all route or used to discover backend buckets.

```sh
s3proxy validate --config ./config.hcl
s3proxy routes --config ./config.hcl
s3proxy serve --config ./config.hcl
```

In a separate client shell with the same private client variables exported:

```sh
export AWS_ACCESS_KEY_ID="$S3PROXY_CLIENT_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$S3PROXY_CLIENT_SECRET_KEY"
export AWS_DEFAULT_REGION=us-east-1
unset AWS_SESSION_TOKEN AWS_SECURITY_TOKEN
aws --endpoint-url http://127.0.0.1:8080 s3api list-buckets
aws --endpoint-url http://127.0.0.1:8080 s3api head-bucket --bucket images
aws --endpoint-url http://127.0.0.1:8080 s3api put-object \
  --bucket images --key hello.txt --body ./hello.txt
aws --endpoint-url http://127.0.0.1:8080 s3api list-objects-v2 --bucket images
```

Create `hello.txt` before uploading. Use ordinary single-request uploads; multipart
and AWS streaming payload formats are unsupported. See [request examples](request-examples.md)
for client compatibility and object read/delete commands.

### Starter Bounds And Timeouts

| Setting                           | Printed value         | Scope                                                   |
| --------------------------------- | --------------------- | ------------------------------------------------------- |
| `replay_body_max_bytes`           | `33554432` (32 MiB)   | Per request when replay buffering is needed             |
| `replay_body_aggregate_max_bytes` | `268435456` (256 MiB) | Retained replay payloads across the process             |
| `timeouts.read_header`            | `10s`                 | Request headers                                         |
| `timeouts.read`                   | `2m`                  | Entire inbound request, including upload body           |
| `timeouts.write`                  | `5m`                  | Response-write deadline, including handler/backend work |
| `timeouts.idle`                   | `60s`                 | Keep-alive idle connection                              |
| target `timeout`                  | `2m`                  | Complete upstream request and response stream           |

These are bounded starter settings for modest objects, not guarantees for every
size or network. Concrete signed payload hashes and unknown-length uploads can
require replay even with one destination. Per-request overflow returns
`413 EntityTooLarge`; aggregate exhaustion returns `503 SlowDown`. Known-length
unsigned payloads may stream without replay, so the replay bound is not a global
upload-size cap or a total process-memory cap. Size the bounds for memory and
concurrency, and align read/write/target and reverse-proxy timeouts for your slowest
expected uploads and downloads. The target timeout covers the entire download,
not just connection establishment.

For containers, follow [the Docker starter workflow](deployment.md#authenticated-starter-in-docker):
override the serve-specific entrypoint for offline commands, change the container
listener to `:8080`, and publish only a restricted host interface. SigV4 provides
authentication, not transport encryption; use the [TLS/reverse-proxy guidance](deployment.md#reverse-proxying)
before exposing the service beyond local trusted access.

## Manual Auth-None Walkthrough

The remaining walkthrough builds from source and uses a hand-written config with
inbound authentication disabled. It is distinct from the printed authenticated
starter and is intended for a trusted test environment.

## Before You Start

You need:

- an S3-compatible backend such as MinIO
- backend credentials with access to a bucket you want to expose
- a local checkout of this repo
- Go 1.26 or newer and `make`

## Install

Build from source:

```sh
make build
./dist/s3proxy version
```

## Minimal Config

Create `config.hcl`:

```hcl
listener "http" "public" {
  address = ":8080"

  addressing {
    path_style     = true
    virtual_hosted = false
  }
}

auth "main" {
  mode = "none"
}

credential "static" "primary" {
  access_key = env("S3PROXY_TARGET_PRIMARY_ACCESS_KEY")
  secret_key = env("S3PROXY_TARGET_PRIMARY_SECRET_KEY")
}

target "s3" "primary" {
  endpoint         = env("S3PROXY_TARGET_PRIMARY_ENDPOINT")
  region           = "us-east-1"
  force_path_style = true
  credentials      = "primary"
}

parser "path_prefix" "images" {
  prefix = "/images"
}

route "images_rw" {
  parser          = "images"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "ListObjectsV2"]
  destinations    = ["primary"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    bucket            = "images-store"
  }
}

bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
```

With that config, requests for `/images/...` are rewritten into the backend bucket `images-store`.

## Export Secrets And Validate

If your config uses `env("...")`, export those variables before running the CLI:

```sh
export S3PROXY_TARGET_PRIMARY_ENDPOINT=http://127.0.0.1:9000
export S3PROXY_TARGET_PRIMARY_ACCESS_KEY=minioadmin
export S3PROXY_TARGET_PRIMARY_SECRET_KEY=minioadmin
```

If you keep them in `.env`, load them first:

```sh
set -a; . ./.env; set +a
```

Validate the config:

```sh
s3proxy validate --config ./config.hcl
```

Start the proxy:

```sh
s3proxy serve --config ./config.hcl
```

From source, the equivalent command is:

```sh
go run ./cmd/s3proxy serve --config ./config.hcl
```

## Send Requests

For trusted local testing with `mode = "none"`, the proxy skips inbound authentication. AWS CLI still needs credentials locally, so provide any placeholder values:

```sh
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

Upload an object through the proxy:

```sh
aws --endpoint-url http://127.0.0.1:8080 s3api put-object \
  --bucket images \
  --key hello.txt \
  --body ./hello.txt
```

List objects through the same route:

```sh
aws --endpoint-url http://127.0.0.1:8080 s3api list-objects-v2 \
  --bucket images
```

List the virtual buckets exposed by the proxy:

```sh
aws --endpoint-url http://127.0.0.1:8080 s3api list-buckets
```

## Switch To SigV4 Auth

For anything outside a trusted environment, use `sigv4_static` so the proxy verifies the caller's S3 SigV4 signature.

```hcl
auth "main" {
  mode = "sigv4_static"

  client "local-dev" {
    access_key      = env("S3PROXY_CLIENT_ACCESS_KEY")
    secret_key      = env("S3PROXY_CLIENT_SECRET_KEY")
    allow_routes    = ["route.images_rw"]
    visible_buckets = ["images"]
  }
}
```

Then point your S3 client or AWS CLI at the proxy with those client credentials:

```sh
export AWS_ACCESS_KEY_ID=$S3PROXY_CLIENT_ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=$S3PROXY_CLIENT_SECRET_KEY
```

The client credentials used to call the proxy are separate from the backend credentials used by `target "s3"`.

## Next Steps

- Add more routes and rewrites in [Configuration](./configuration.md)
- Set up fan-out replication or failover in [Config Examples](./config-examples.md)
- Review exact behavior in [API Reference](./api-reference.md)
