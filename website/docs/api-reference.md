---
sidebar_position: 5
---

# API Reference

`s3proxy` exposes an S3-compatible HTTP API rather than a custom JSON API.

This page focuses on the proxy-facing contract, supported S3 operations, and the behavior that is specific to the proxy.

## Supported Operations

| Operation                   | Supported | Notes                                |
| --------------------------- | --------- | ------------------------------------ |
| `GetObject`                 | Yes       | Reads from one effective destination |
| `HeadObject`                | Yes       | Reads from one effective destination |
| `PutObject`                 | Yes       | Can fan out with `dispatch = "all"`  |
| `DeleteObject`              | Yes       | Can fan out with `dispatch = "all"`  |
| `HeadBucket`                | Yes       | Route-selected backend               |
| `ListObjectsV2`             | Yes       | Uses one effective backend only      |
| `ListBuckets`               | Yes       | Proxy-defined virtual bucket list    |
| `CopyObject`                | No        | Returns `NotImplemented`             |
| Multipart upload operations | No        | Return `NotImplemented`              |

## Addressing Modes

The proxy accepts:

- path-style addressing
- virtual-hosted addressing

The listener decides which forms are enabled.

## Authentication Modes

Supported inbound auth modes:

- `none`
- `sigv4_static`

With `sigv4_static`, the proxy verifies the inbound S3 SigV4 signature against statically configured clients.

## Request Classification

Routing uses S3 operation classification, not only the HTTP method.

Examples:

- `GET /bucket/key` can classify as `GetObject`
- `HEAD /bucket` can classify as `HeadBucket`
- `GET /?list-type=2` can classify as `ListObjectsV2`

That classification is what route `operations = [...]` filters use.

## Routing-Specific Behavior

Routes are evaluated in config order and may stop or continue after a match.

Important behavior:

- `dispatch = "first"` uses one destination
- `dispatch = "all"` replays supported writes to all destinations
- reads never fan out in v1
- `read_preference` chooses the effective backend for reads when multiple destinations are configured

## `ListBuckets`

`ListBuckets` is virtual.

The proxy returns buckets defined in `bucket` blocks and filtered by the authenticated client's `visible_buckets` policy. It does not call the backend to discover buckets.

## `ListObjectsV2`

`ListObjectsV2` is forwarded to one selected backend.

The proxy does not merge listing results or pagination tokens across multiple backends in v1.

## Failover Rules

For `read_preference = "ordered_failover"`, failover happens only on:

- transport errors
- request timeouts
- upstream `5xx`

Failover does not happen on:

- `404`
- `NoSuchKey`
- `NoSuchBucket`

## Fan-Out Writes

For `dispatch = "all"`:

- `PutObject` is supported
- `DeleteObject` is supported
- the request body is buffered in memory so it can be replayed, bounded by `listener.replay_body_max_bytes` per request and `listener.replay_body_aggregate_max_bytes` across the process
- if any destination fails, the request fails overall
- upstream HTTP failures preserve the primary upstream error response when available
- transport or replay failures return a proxy-generated failure
- oversized replay attempts fail with `413 EntityTooLarge`; aggregate replay-budget exhaustion fails with `503 SlowDown`

For writes matched by multiple routes through `on_match = "continue"`, every matched route must also succeed. A later route failure is returned as failure rather than hiding behind an earlier success.

## Outbound Signing

The proxy terminates and rebuilds requests before forwarding them.

Outbound S3 requests are signed with the destination backend credentials, not the inbound client credentials.

For outbound SigV4, the proxy uses `UNSIGNED-PAYLOAD`, and `Content-Length` must be set before signing.

## Error Behavior

Representative error rules:

- unsupported operations return S3-compatible `NotImplemented`
- route misses return standard S3-compatible error responses
- upstream backend failures propagate as proxy-mediated S3 responses
- multi-destination write failures are surfaced as failures, not partial success
- multi-route write failures are surfaced as failures, not partial success
