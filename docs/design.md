# S3 Proxy Design

## Overview

This service is a Go-based S3-compatible proxy that accepts client S3 API requests and forwards them to one or more configured S3-compatible backends.

The proxy determines the destination backend from the incoming request path, bucket name, host, and operation type. Before forwarding, it may rewrite the bucket and key path, then signs the outbound request using the destination backend's credentials.

The service is delivered as:

- a single Go CLI binary
- a container image exposing the service as a public HTTP API

Configuration is written in HCL with an Alloy-like labeled block style.

## Goals

- Accept S3-compatible API requests from clients.
- Optionally authenticate clients with SigV4.
- Route requests based on path, bucket, host, and operation.
- Forward requests to one or more S3-compatible backends.
- Rewrite bucket names, key prefixes, and key templates before forwarding.
- Support one or multiple destination backends per route.
- Support route matching modes:
  - stop on first match
  - continue collecting matches
- Support destination dispatch modes:
  - first destination only
  - all destinations
- Support virtual `ListBuckets`.
- Keep `ListObjects*` compatible with S3 pagination by selecting one effective backend in v1.
- Package the service as a single binary and Docker image.

## Non-Goals For V1

- Dynamic client credential issuance
- Database-backed credential storage
- Multipart upload support
- Merged multi-backend `ListObjects` pagination
- Non-S3 backend types
- Full IAM policy emulation
- Hot config reload
- Full presigned URL feature parity

## Core Design Principle

The proxy terminates and rebuilds the request. It is not a blind relay.

Reason:

- S3 signatures cover host, path, query, and headers.
- The proxy may rewrite bucket and key paths.
- Destination backends use different credentials.
- Therefore, the inbound signature cannot be forwarded unchanged.

The proxy must:

1. Parse and normalize the inbound request.
2. Authenticate and authorize the client if enabled.
3. Determine the route and destination(s).
4. Rewrite bucket/key/path as required.
5. Build outbound request(s).
6. Sign outbound request(s) with destination credentials.
7. Return a compatible S3 response to the client.

## Request Model

### Supported Addressing Modes

The proxy supports:

- path-style requests: `/bucket/key`
- virtual-hosted requests: `bucket.proxy.example.com/key`

### Normalized Request Context

All requests should be normalized into an internal request context:

```go
type RequestContext struct {
    Host           string
    RawPath        string
    Bucket         string
    Key            string
    Query          url.Values
    Method         string
    Headers        http.Header
    Operation      S3Operation
    AddressingMode AddressingMode
    Captures       map[string]string
}
```

### Operation Classification

Routing should use an operation classifier, not just the HTTP method.

Initial operation support for v1:

- `GetObject`
- `HeadObject`
- `PutObject`
- `DeleteObject`
- `HeadBucket`
- `ListObjectsV2`
- `ListBuckets`

Multipart-related operations are explicitly unsupported in v1 and should return an S3-compatible `NotImplemented` error.

## Authentication And Authorization

### Supported Modes

Two auth modes are supported:

- `none`
- `sigv4_static`

### `none`

No inbound authentication is performed. This mode is intended for trusted deployments only.

### `sigv4_static`

The proxy validates the inbound S3 SigV4 signature using statically configured client credentials from HCL.

This mode does not require a database.

### Client Authorization

After successful authentication, the client can be authorized by:

- allowed routes
- optionally allowed operations
- visible virtual buckets for `ListBuckets`

Inbound client credentials and destination backend credentials are fully separate.

### Deferred Auth Features

The following are deferred to a later phase:

- credential generation
- credential rotation
- credential revocation
- admin API
- DB-backed client store

## Routing Model

### Route Evaluation

Routes are evaluated in config order.

Each route contains:

- a parser reference
- optional operation filters
- one or more destinations
- `on_match`
- `dispatch`
- `read_preference`
- rewrite rules

### `on_match`

Controls route evaluation after a match:

- `stop`
- `continue`

### `dispatch`

Controls how matched destinations are used:

- `first`
- `all`

### `read_preference`

Controls read target selection when a route has multiple destinations:

- `first`
- `random`
- `hash`
- `ordered_failover`

In v1, reads always go to one effective backend only.

### Failover Policy

For `ordered_failover`, failover happens only on:

- transport errors
- timeouts
- upstream `5xx` errors

Failover does not happen on:

- `404`
- `NoSuchKey`
- `NoSuchBucket`

This avoids hiding routing mistakes or backend inconsistency.

## Parser Model

Initial parser types:

- `parser.path_prefix`
- `parser.bucket_exact`
- `parser.bucket_regex`
- `parser.host_suffix`

Parsers extract routing context and may capture named regex groups for rewrites.

Examples:

- `/images/cat.jpg`
- bucket `tenant-acme-logs`
- host suffix `s3proxy.example.com`

## Rewrite Model

Rewrites are applied after route selection and before outbound request construction.

Initial rewrite fields:

- `strip_path_prefix`
- `strip_key_prefix`
- `prepend_key_prefix`
- `bucket`
- `key_template`

Examples:

- strip `/images` from the request path
- prepend `assets/` to the resulting key
- rewrite bucket `tenant-acme-logs` to `shared-logs`
- rewrite key as `{{ .Captures.tenant }}/{{ .Key }}`

Template fields use capitalized Go names: `Bucket`, `Key`, and `Captures`
(a `map[string]string` populated from regex named-group captures).

## Backend Model

### Initial Backend Type

Only one backend type is supported in v1:

- `target.s3`

Each S3 target includes:

- endpoint URL
- region
- credentials
- path-style option
- transport settings
- timeout settings

## Forwarding Behavior

### Single Destination

For `dispatch = "first"`:

- build one outbound request
- apply rewrite
- sign with target credentials
- return the upstream response

### Multi-Destination Writes

For `dispatch = "all"` in v1:

- supported only for basic single-request write operations
- initially:
  - `PutObject`
  - `DeleteObject`

Success policy:

- all destinations must succeed
- if any destination fails, return failure to the client
- log per-destination result details

Reads do not fan out in v1.

## Listing Semantics

### `ListBuckets`

`ListBuckets` is supported as a proxy-defined virtual view.

It does not aggregate actual upstream bucket listings.

Instead, the proxy returns buckets explicitly exposed in config and visible to the authenticated client.

This keeps the behavior:

- predictable
- secure
- independent of backend-specific bucket visibility

### `ListObjectsV2`

`ListObjectsV2` is supported only against one effective backend in v1.

If a route has multiple destinations, `read_preference` selects the single backend used for the list call.

The proxy does not merge list results or pagination state across multiple backends in v1.

Reason:

- S3 clients expect stable continuation semantics
- global lexicographic merge across backends is complex
- merged tokens require custom opaque cursor state

## Multipart Uploads

Multipart upload support is deferred from v1.

All multipart-related requests should return an S3-compatible XML error:

- HTTP status: `501 Not Implemented`
- S3 error code: `NotImplemented`

Reason:

- multipart fan-out requires persistent upload ID mapping
- each backend returns its own `UploadId`
- the proxy would need to track client upload IDs to backend upload IDs

A later implementation will require a persistent state store.

## Error Handling

The proxy should return S3-compatible XML errors where possible.

Typical cases:

- auth failure
- unknown route
- invalid rewrite result
- unsupported operation
- upstream failure
- fan-out partial or full failure

Rules:

- config errors fail startup
- unsupported multipart returns `NotImplemented`
- route resolution failures return a compatible S3 error
- proxy-generated errors should include request ID for debugging

## Config Model

Configuration uses Alloy-like labeled HCL blocks.

Recommended block types:

- `listener.http`
- `auth`
- `client`
- `credential.static`
- `target.s3`
- `parser.path_prefix`
- `parser.bucket_exact`
- `parser.bucket_regex`
- `parser.host_suffix`
- `route`
- `bucket`

### Example Config

> **Note on reference syntax**: The proxy uses HCL two-label block syntax
> (e.g. `listener "http" "public" {}`) rather than dotted block names.
> References between blocks are written as quoted strings containing the
> block labels (e.g. `parser = "images"` or `destinations = ["primary"]`).
> The `env("VAR")` call is supported in any string attribute and is replaced
> at load time with the environment variable value.
>
> `key_template` uses `text/template` with `Bucket`, `Key`, and `Captures`
> (a `map[string]string` of regex named-group captures) available on the
> template data. Use capitalized field names: `{{ .Captures.tenant }}/{{ .Key }}`.

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
      "route.logs_read",
    ]

    visible_buckets = [
      "images",
      "tenant-acme-logs",
    ]
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

credential "static" "replica" {
  access_key = env("S3PROXY_TARGET_REPLICA_ACCESS_KEY")
  secret_key = env("S3PROXY_TARGET_REPLICA_SECRET_KEY")
}

target "s3" "primary" {
  endpoint         = "https://minio-a.internal"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "primary"
}

target "s3" "replica" {
  endpoint         = "https://minio-b.internal"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "replica"
}

parser "path_prefix" "images" {
  prefix = "/images"
}

parser "bucket_regex" "tenant_logs" {
  pattern = "^tenant-(?P<tenant>[a-z0-9-]+)-logs$"
}

route "images_rw" {
  parser          = "images"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "ListObjectsV2"]
  destinations    = ["primary"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix  = "/images"
    prepend_key_prefix = "assets/"
    bucket             = "images-store"
  }
}

route "logs_read" {
  parser          = "tenant_logs"
  operations      = ["GetObject", "HeadObject", "ListObjectsV2", "HeadBucket"]
  destinations    = ["primary", "replica"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    bucket       = "shared-logs"
    key_template = "{{ .Captures.tenant }}/{{ .Key }}"
  }
}

route "logs_write" {
  parser       = "tenant_logs"
  operations   = ["PutObject", "DeleteObject"]
  destinations = ["primary", "replica"]
  dispatch     = "all"
  on_match     = "stop"

  rewrite {
    bucket       = "shared-logs"
    key_template = "{{ .Captures.tenant }}/{{ .Key }}"
  }
}

bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}

bucket "tenant_acme_logs" {
  visible_name = "tenant-acme-logs"
  route        = "logs_read"
}
```

## Validation Rules

The config loader should validate:

- duplicate labels
- unknown references
- routes without destinations
- invalid enum values
- invalid parser configuration
- invalid rewrite combinations
- duplicate visible bucket names
- invalid auth configuration
- invalid target configuration

The service should fail startup on invalid config.

## CLI Design

The service is a single binary named `s3proxy`.

Recommended commands:

- `s3proxy serve --config /etc/s3proxy/config.hcl`
- `s3proxy validate --config /etc/s3proxy/config.hcl`
- `s3proxy version`

Optional future commands:

- `s3proxy print-example-config`
- `s3proxy routes --config ...`

## Deployment Model

The service is packaged as a Docker image that runs the CLI.

Recommended container behavior:

- expose the proxy on `:8080`
- mount config at `/etc/s3proxy/config.hcl`
- pass secrets via environment variables

Recommended image approach:

- multi-stage Docker build
- static or near-static Go binary
- minimal runtime image with CA certificates

## Recommended Package Layout

```text
cmd/s3proxy/
internal/app/
internal/config/
internal/auth/
internal/requestctx/
internal/s3ops/
internal/router/
internal/rewrite/
internal/backend/s3/
internal/dispatch/
internal/listbuckets/
internal/httpapi/
internal/observability/
internal/xmls3/
```

## Implementation Plan

### Milestone 1

- CLI scaffold
- HCL config parsing and validation
- request normalization
- operation classification
- auth modes:
  - `none`
  - `sigv4_static`
- parser and route matching
- rewrite engine
- single-destination forwarding for:
  - `GetObject`
  - `HeadObject`
  - `PutObject`
  - `DeleteObject`
  - `HeadBucket`
  - `ListObjectsV2`
- virtual `ListBuckets`
- logs, health, readiness, basic metrics

### Milestone 2

- multi-destination fan-out for:
  - `PutObject`
  - `DeleteObject`
- stronger failure handling
- more metrics and logging detail
- more integration tests

### Later Phase

- multipart upload support with persistent state store
- DB-backed client management
- credential issuance, rotation, revocation
- merged multi-backend listing if ever needed
- broader presigned URL handling
- hot config reload

## Testing Strategy

### Unit Tests

- config parsing and validation
- request parsing
- operation classification
- auth validation
- route matching
- rewrite behavior
- read preference selection
- virtual `ListBuckets` response rendering

### Integration Tests

Run against local S3-compatible services such as MinIO:

- authenticated `GetObject`
- authenticated `PutObject`
- invalid signature rejection
- path-style routing
- virtual-hosted routing
- rewrite behavior
- `ListObjectsV2`
- virtual `ListBuckets`
- multi-destination write success/failure behavior

### Explicit Unsupported Tests

- multipart requests return `NotImplemented`
- merged multi-backend list behavior is rejected or absent

## Deferred Features

The following are intentionally out of scope for v1:

- multipart upload support
- client credential generation
- DB-backed auth management
- merged multi-backend `ListObjects` pagination
- advanced presigned URL compatibility
- live config reload

## Appendix: Open Questions And Rejected Alternatives

### Open Questions

- Should inbound presigned URL support be part of the first implementation, or explicitly unsupported until rewrite and host-handling semantics are better defined?
- Should route authorization remain route-centric only, or should the config eventually support finer-grained per-bucket and per-prefix authorization rules?
- Should the service support live config reload in a later phase, or keep restart-based deploys as the operational model?
- Should later multipart support use an embedded local store such as SQLite, or start directly with a shared external store for multi-instance deployments?

### Rejected Or Deferred Alternatives

#### DB-backed client credential management in v1

Rejected for v1.

Reason:

- inbound SigV4 validation works with static credentials
- adding a DB early would increase operational and implementation complexity
- credential issuance, rotation, and revocation are better handled in a later phase with a clear admin model

#### Real upstream `ListBuckets` aggregation

Rejected for v1.

Reason:

- it leaks backend bucket topology
- different destinations may expose unrelated namespaces
- it is hard to make secure and predictable across multiple backends

The chosen design is proxy-defined virtual buckets.

#### Merged multi-backend `ListObjects*`

Rejected for v1.

Reason:

- S3 clients expect stable pagination semantics
- merged listing requires ordering guarantees across backends
- continuation tokens would need proxy-owned opaque state

The chosen design is to select one effective read backend per request.

#### Read fan-out across all matching destinations

Rejected for v1.

Reason:

- it complicates response semantics and latency behavior
- it is especially problematic for list operations
- it can hide routing and consistency issues

The chosen design is single effective read routing with configurable `read_preference`.

#### Multipart uploads in v1

Deferred.

Reason:

- fan-out multipart uploads require persistent mapping between client upload IDs and backend upload IDs
- even single-backend multipart support adds protocol surface area beyond the initial single-request operations

The chosen v1 behavior is S3-compatible `NotImplemented`.

#### Failing over on `404` / `NoSuchKey` / `NoSuchBucket`

Rejected.

Reason:

- this can mask routing mistakes
- this can hide replication lag or backend inconsistency
- it makes read behavior less predictable for clients

The chosen design limits `ordered_failover` to transport errors, timeouts, and upstream `5xx` responses.

## Final V1 Decisions

- inbound authentication is optional
- authenticated mode uses static SigV4 client credentials from config
- destination credentials are separate from client credentials
- `ListBuckets` is a proxy-defined virtual list
- reads always use one effective backend in v1
- `ordered_failover` only retries on transport errors, timeouts, and `5xx`
- multipart is deferred and returns S3-compatible `NotImplemented`
- credential generation is deferred to a later DB-backed phase
