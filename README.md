# s3proxy

A path-based multi-backend proxy for S3-compatible APIs, written in Go.

The proxy accepts S3-compatible requests, optionally authenticates clients with
SigV4, determines one or more destination backends from the incoming request
path, bucket name, host, and operation type, rewrites requests as needed, and
forwards them to S3-compatible targets using the destinations' credentials.

## Current V1 Scope

### Supported Operations

- `GetObject`
- `HeadObject`
- `PutObject`
- `DeleteObject`
- `HeadBucket`
- `ListObjectsV2`
- `ListBuckets` (virtual/proxy-defined)

### Auth Modes

- `none` – skip inbound authentication (trusted environments only)
- `sigv4_static` – validate inbound SigV4 signatures with statically configured client credentials

### Routing

- path-style and virtual-hosted addressing
- parsers: path prefix, bucket exact, bucket regex, host suffix
- ordered route evaluation with `stop` / `continue` match modes
- destination dispatch modes: `first` (single target) and `all` (fan-out)
- read preferences: `first`, `random`, `hash`, `ordered_failover`

### Rewrite

- strip path prefix
- strip / prepend key prefix
- override bucket name
- key templates with captures from regex parsers

### Unsupported in V1

- Multipart uploads — return S3-compatible `NotImplemented`
- Credential generation / rotation / DB-backed auth
- Merged multi-backend `ListObjects` pagination
- Hot config reload
- Full presigned URL feature parity

See [docs/design.md](docs/design.md) for the full design document.

## Install

### via asdf

Add the plugin:

```sh
asdf plugin add s3proxy
# or
asdf plugin add s3proxy https://github.com/egose/s3proxy.git
```

Install and activate a version:

```sh
# List all available versions
asdf list all s3proxy

# Install a specific version
asdf install s3proxy <version>

# Install the latest stable version
asdf install s3proxy latest

# Set the global version
asdf global s3proxy <version>
```

Once installed, the `s3proxy` binary is available directly on your `PATH`:

```sh
s3proxy serve --config /etc/s3proxy/config.hcl
s3proxy validate --config /etc/s3proxy/config.hcl
s3proxy version
```

Please check the [asdf documentation](https://github.com/asdf-vm/asdf) for more details.

## Example Configuration

```hcl
listener "http" "public" {
  address = ":8080"

  addressing {
    path_style     = true
    virtual_hosted = true
    host_suffixes  = ["s3proxy.example.com"]
  }

  timeouts {
    read_header = "10s"
    idle        = "60s"
    write       = "0s"
  }
}

auth "main" {
  mode = "sigv4_static"

  client "ci" {
    access_key = env("S3PROXY_CLIENT_CI_ACCESS_KEY")
    secret_key = env("S3PROXY_CLIENT_CI_SECRET_KEY")

    allow_routes = [
      "route.images_rw",
    ]

    visible_buckets = ["images"]
  }

  client "admin" {
    access_key = env("S3PROXY_CLIENT_ADMIN_ACCESS_KEY")
    secret_key = env("S3PROXY_CLIENT_ADMIN_SECRET_KEY")

    allow_routes    = ["*"]
    visible_buckets = ["*"]
  }
}

credential "static" "primary" {
  access_key = env("S3PROXY_TARGET_PRIMARY_ACCESS_KEY")
  secret_key = env("S3PROXY_TARGET_PRIMARY_SECRET_KEY")
}

target "s3" "primary" {
  endpoint         = "https://minio-a.internal"
  region           = "us-east-1"
  force_path_style = true
  timeout          = "5s"
  credentials      = "primary"
}

parser "path_prefix" "images" {
  prefix = "/images"
}

parser "bucket_regex" "tenant_logs" {
  pattern = "^tenant-(?P<tenant>[a-z0-9-]+)-logs$"
}

route "images_rw" {
  parser        = "images"
  operations    = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "ListObjectsV2"]
  destinations  = ["primary"]
  dispatch      = "first"
  on_match      = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix  = "/images"
    prepend_key_prefix = "assets/"
    bucket             = "images-store"
  }
}

bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
```

## CLI

```sh
s3proxy serve --config /etc/s3proxy/config.hcl
s3proxy validate --config /etc/s3proxy/config.hcl
s3proxy version
```

## Docker

```sh
docker build -t s3proxy .
docker run --rm \
  -p 8080:8080 \
  -v ./config.hcl:/etc/s3proxy/config.hcl:ro \
  -e S3PROXY_CLIENT_CI_ACCESS_KEY=... \
  -e S3PROXY_CLIENT_CI_SECRET_KEY=... \
  -e S3PROXY_TARGET_PRIMARY_ACCESS_KEY=... \
  -e S3PROXY_TARGET_PRIMARY_SECRET_KEY=... \
  s3proxy
```

## Local Run

Create `config.hcl` then:

```sh
go run ./cmd/s3proxy serve --config config.hcl
```

## Tests

```sh
go test ./...                                  # unit tests
make vet test                                  # vet + unit tests
make test-race                                # unit tests with the race detector
```

### Integration tests

The integration test suite (`internal/integration`, build-tagged `integration`)
exercises the proxy end-to-end against the sandbox docker-compose stack:
MinIO as the primary backend and SeaweedFS as the replica.

```sh
cp .env.example .env                            # then edit secrets if you want
make sandbox-integration-up                    # start stack + proxy + run tests + tear down
# or, to leave the stack up for iterative runs:
make sandbox-up DAEMON=true
make build
make test-integration                          # repeats against the running stack
make sandbox-down
make test-integration-race                      # fan-out path under the race detector
```

`make sandbox-integration-up` sources `.env` into the proxy's environment
(so `env("...")` config substitution resolves), runs the stack + proxy +
tests + teardown in one shot, and exits with the test's status.

See `sandbox/integration-config.hcl` for the routing/parsers/credentials
configuration the suite expects.

## Environment Variables

Use `env("VAR")` in any string attribute in the HCL config to inline an
environment variable. This is necessary for secrets — do not commit secret
values into the config file.

## Notes on Behavior

- `key_template` uses Go template syntax. Accessed data is keyed by `Bucket`,
  `Key`, and `Captures` (a `map[string]string` of regex named-group captures).
- Escaped path bytes are preserved through routing and rewrites, so keys such
  as `%2F` stay distinct from literal path separators.
- For multi-destination routes, `read_preference` controls which destination
  is used for reads (`first` by default).
- For multi-destination writes with `dispatch = "all"`, all destinations must
  succeed; any failure returns a 502 to the client.
- `ordered_failover` only fails over on transport errors, timeouts, and
  upstream `5xx`. It does not fail over on `404` / `NoSuchKey` /
  `NoSuchBucket` responses.
- `target "s3"` supports an optional `timeout` duration. For
  `ordered_failover`, this directly bounds how long the proxy waits before
  failing over from a slow or unreachable backend.
- `ListBuckets` returns proxy-defined virtual buckets, not upstream discovery.
- Request body fan-out for multi-destination writes reads the entire body into
  memory before replaying to each destination.

## Deferred / Planned

See the "Deferred Features" section in [docs/design.md](docs/design.md) for a
full list of intentionally deferred features, including multipart upload
support and dynamic credential generation.
