---
sidebar_position: 1
---

# s3proxy

`s3proxy` is a Go service that accepts S3-compatible requests and forwards them to one or more configured S3-compatible backends.

It sits at the S3 protocol boundary: the proxy can authenticate the client, classify the S3 operation, match a route from the request path, bucket name, host, and operation, rewrite the bucket or key, then sign outbound requests with backend credentials.

## What It Solves

`s3proxy` is useful when you want one stable S3 endpoint while the actual storage layout behind it is more opinionated than a single bucket on a single backend.

Typical reasons to use it:

- expose path-based or host-based virtual views over existing buckets
- replicate writes to more than one S3-compatible backend
- fail reads over to a secondary backend on transport errors and upstream `5xx`
- keep client credentials separate from backend credentials
- present a proxy-defined `ListBuckets` view instead of exposing upstream bucket discovery

## Current V1 Scope

Supported operations:

- `GetObject`
- `HeadObject`
- `PutObject`
- `DeleteObject`
- `HeadBucket`
- `ListObjectsV2`
- `ListBuckets`

Supported inbound auth modes:

- `none`
- `sigv4_static`

Supported routing and rewrite features:

- path-style and virtual-hosted addressing
- parsers: `path_prefix`, `bucket_exact`, `bucket_regex`, `host_suffix`
- ordered route evaluation with `stop` and `continue`
- destination dispatch modes: `first` and `all`
- read preferences: `first`, `random`, `hash`, `ordered_failover`
- rewrite rules: `strip_path_prefix`, `strip_key_prefix`, `prepend_key_prefix`, `bucket`, `key_template`

## Non-Goals In V1

The project intentionally does not implement these yet:

- multipart upload support
- merged multi-backend `ListObjects` pagination
- credential generation or rotation
- non-S3 backend types
- hot config reload
- full presigned URL feature parity

Multipart-related operations and `CopyObject` return an S3-compatible `NotImplemented` error.

## How A Request Flows

1. The proxy receives an S3-compatible HTTP request.
2. It derives a normalized request context from host, path, query, and method.
3. If auth is enabled, it validates the inbound SigV4 signature against configured client credentials.
4. It classifies the operation and resolves one or more matching routes.
5. It applies any configured bucket or key rewrites.
6. It builds outbound S3 requests to the selected target backends.
7. It signs those outbound requests with the target credentials and returns an S3-compatible response.

## Key Behaviors

- `ListBuckets` is virtual. It returns proxy-defined buckets visible to the authenticated client, not upstream bucket discovery.
- Reads never fan out in v1. Even when a route has multiple destinations, one effective backend is selected per read.
- For `dispatch = "all"`, write request bodies are buffered in memory before they are replayed to each destination, bounded by `listener.replay_body_max_bytes`.
- `ordered_failover` only fails over on transport errors, timeouts, and upstream `5xx`. It does not fail over on `404`, `NoSuchKey`, or `NoSuchBucket`.
- `path_prefix` matching is strict: `RawPath == prefix` or `RawPath` starts with `prefix + "/"`.

## Documentation Map

- [Quickstart](./quickstart.md) for a first local setup
- [Configuration](./configuration.md) for the HCL model and validation rules
- [Config Examples](./config-examples.md) for complete sample configs
- [Providers and Routing](./providers-and-routing.md) for route evaluation, parsers, rewrites, and failover
- [Request Examples](./request-examples.md) for AWS CLI examples against the proxy
- [API Reference](./api-reference.md) for operation coverage and request behavior
- [Operations](./operations.md) for build, test, sandbox, and runtime commands
- [Deployment](./deployment.md) for Docker, service management, and rollout guidance

## Design Document

For the full implementation rationale and deferred feature list, see the repository design doc:

- [github.com/egose/s3proxy/blob/main/docs/design.md](https://github.com/egose/s3proxy/blob/main/docs/design.md)
