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
| `ListObjectsV1`             | No        | Returns `NotImplemented`             |
| `ListBuckets`               | Yes       | Proxy-defined virtual bucket list    |
| `CopyObject`                | No        | Returns `NotImplemented`             |
| Multipart upload operations | No        | Return `NotImplemented`              |

## Supported Query Keys

The v1 API rejects malformed query strings with `400 InvalidRequest` before authentication, request-body reads, or route dispatch. Invalid/truncated percent escapes in names or values and unescaped semicolons are malformed; the proxy never executes the successfully decoded subset of such a query. Error responses and logs do not include raw query values or parser excerpts.

Well-formed queries with keys outside the supported operation contract return `501 NotImplemented` before route dispatch, subject to normal authentication checks. For example, `?versionId=%GG` and `?tagging=;value` return 400, while `?versionId=value` and `?tagging=%3Bvalue` are well-formed unsupported selectors and return 501. Encoded punctuation (`%3B`, `%25`, `%2B`), spaces, Unicode, and opaque continuation tokens retain normal query-decoding behavior. Existing operation-specific duplicate-key rules still apply.

Inbound SigV4 presign query keys are accepted for authentication and are not forwarded to backends: `X-Amz-Algorithm`, `X-Amz-Credential`, `X-Amz-Date`, `X-Amz-Expires`, `X-Amz-Security-Token`, `X-Amz-Signature`, and `X-Amz-SignedHeaders`.

| Operation                                                                           | Supported non-auth query keys                                                                                                                            |
| ----------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GetObject`, `HeadObject`, `PutObject`, `DeleteObject`, `HeadBucket`, `ListBuckets` | none, except optional AWS SDK `x-id` matching the operation name                                                                                         |
| `ListObjectsV2`                                                                     | `list-type=2`, `continuation-token`, `delimiter`, `encoding-type`, `fetch-owner`, `max-keys`, `prefix`, `start-after`, and optional `x-id=ListObjectsV2` |

Unsupported query operations include `acl`, `tagging`, `retention`, `legal-hold`, `torrent`, `versioning`, `versions`, `versionId`, `restore`, `select`, response header overrides such as `response-content-type`, and multipart variants such as `uploads`, `uploadId`, and `partNumber`.

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

Header signatures, presigned URLs, and outbound signatures use S3 canonical URI rules, preserving the escaped path without double escaping or path normalization. Spaces, percent characters, Unicode, and escaped separators in keys are supported. This corrects the previous generic-signer behavior: custom clients using the standalone AWS SDK v2 signer must set `DisableURIPathEscaping = true`. Signatures that double-escape the path are rejected.

Header-signed requests must sign `x-amz-date` and be within 15 minutes of the proxy clock. Presigned URLs may expire at most seven days after signing. Payload hashes may be omitted, use `UNSIGNED-PAYLOAD`, or contain a 64-character hexadecimal SHA-256 digest. Unsupported streaming formats are rejected as described below.

## Request Payload Formats

Use ordinary single-request `PutObject` uploads. The proxy does not decode AWS streaming envelopes, verify per-chunk signatures, or process S3 checksum trailers. Requests declaring any of the following return S3 `501 NotImplemented`:

- a comma-separated `Content-Encoding` token equal to `aws-chunked` (case-insensitive, with surrounding whitespace ignored)
- an `X-Amz-Content-Sha256` value starting with `STREAMING-` (case-insensitive, with surrounding whitespace ignored), including signed-payload and unsigned-trailer sentinels
- a nonempty `X-Amz-Trailer` value after trimming surrounding whitespace

Header names are case-insensitive and every repeated value is checked. Encoding matching is by whole token, not substring: `not-aws-chunked` does not trigger this boundary. The guard only inspects metadata; it neither reads nor rewrites the object body or request headers.

This format rejection occurs after strict query validation and before authentication, payload-hash verification, replay buffering, routing, or backend I/O, in both `none` and `sigv4_static` modes (header signatures and presigned URLs). **This changes streaming hash sentinels from their former authentication failure to consistent `501 NotImplemented`.** Malformed queries still return `400 InvalidRequest` first. Eligible local health probes retain their existing behavior. Errors and logs do not echo the rejected marker values.

Ordinary HTTP `Transfer-Encoding: chunked`, gzip-encoded object bytes with `Content-Encoding: gzip`, regular checksum headers, and opaque bytes that resemble AWS chunks are not rejected by this boundary. Ordinary authentication, operation, and replay limits still apply, including `413 EntityTooLarge` and `503 SlowDown` where buffering is required. A regular checksum header is distinct from requesting unsupported trailer processing; this boundary does not itself validate checksum values.

## Request Classification

Routing uses S3 operation classification, not only the HTTP method.

Examples:

- `GET /bucket/key` can classify as `GetObject`
- `HEAD /bucket` can classify as `HeadBucket`
- `GET /bucket?list-type=2` can classify as `ListObjectsV2`

That classification is what route `operations = [...]` filters use.

## Routing-Specific Behavior

Routes are evaluated in config order and may stop or continue after a match.

Important behavior:

- `dispatch = "first"` uses one destination
- `dispatch = "all"` replays supported writes to all destinations
- reads never fan out in v1
- `read_preference` chooses the effective backend for reads when multiple destinations are configured

## Streaming Responses

Object downloads stream from the selected backend. If copying a response fails,
the proxy aborts the transfer: clients receive a request or body-read error
rather than normal completion, including for unknown-length bodies. Headers
already received may still show `200 OK`; clients must read the entire body
successfully before treating the download as complete. The proxy never appends
an XML error to an interrupted object body.

The same transfer-failure behavior applies when forwarding upstream error XML
or transformed listing XML. A body-close error after a complete copy is logged
as a cleanup failure and does not interrupt successful delivery. Completion
logs retain the committed status and bytes accepted by the response writer;
check for `response copy failed` to identify aborted transfers.

## `ListBuckets`

`ListBuckets` is virtual.

The proxy returns buckets defined in `bucket` blocks and filtered by the authenticated client's `visible_buckets` policy. It does not call the backend to discover buckets or resolve a route. Adding `ListBuckets` to a route's operations therefore has no routing effect.

## `ListObjectsV2`

`ListObjectsV2` is forwarded to one selected backend.

The proxy does not merge listing results or pagination tokens across multiple backends in v1.

Bucket-only rewrites, prepend prefixes, and prefix-only templates translate
`prefix` and `start-after` to the backend namespace. XML `Name`, `Prefix`,
`StartAfter`, `Contents.Key`, and `CommonPrefixes.Prefix` are translated back.
See [rewrite validation](configuration.md#rewrites) for the exact supported
template syntax; strip-key/strip-path rules are unsupported for listings.

`encoding-type=url` is supported. Query parameters contain S3 key values after
normal query decoding; XML key fields are percent-decoded once, translated, and
percent-encoded again. Encoded XML uses `+` or `%20` for space and `%2B` for a
literal plus; output uses `%20` for spaces. Raw path prefixes still treat `+`
literally. Keys returned by a client SDK can be used with GetObject through the same mapping.
Use normal path escaping when constructing raw HTTP requests.

Known query echoes (Prefix/StartAfter/Delimiter) are also accepted when exactly
equal to the unencoded request value, as emitted by SeaweedFS; they are encoded
consistently in the client response. Returned keys/common prefixes must follow
the declared encoding.

Continuation tokens are opaque: the proxy preserves their values without
namespace translation or URL decoding inside XML. `random` selection or
`ordered_failover` can switch backends between pages, so tokens can be rejected
or pages inconsistent. Choose a fixed backend for stable pagination; no cursor
affinity or merged listing is provided. Hash-selected bucket listings are stable
for unchanged destinations, but object reads can hash to another replica.

Successful XML transformation is bounded to 8 MiB input/output, 50,000 elements,
four element levels, and five seconds, with earlier request/target cancellation
taking precedence. Bodies close on every exit. Malformed XML, unsupported XML
structure, encoding mismatches, oversized responses, or keys outside the
requested namespace return HTTP `502` with S3 `InternalError` without partial results. An object
equal to the hidden namespace prefix has no nonempty visible key and is also
rejected. Transformed XML gets fresh framing and integrity headers are removed;
object downloads continue to stream. Standard V2 owner/checksum/restore metadata
is preserved, while unknown XML extensions fail closed.

`HeadBucket` always addresses the rewritten bucket root, ignoring object key
rules. It checks the backend bucket, not whether the namespace prefix exists.

## Failover Rules

For `read_preference = "ordered_failover"`, the proxy tries the next destination on:

- errors returned while preparing, signing, or sending the upstream request, including transport errors, request timeouts, and replay-limit errors
- upstream `5xx`

Failover does not happen on:

- `404`
- `NoSuchKey`
- `NoSuchBucket`
- any other upstream `4xx`

## Fan-Out Writes

For `dispatch = "all"`:

- `PutObject` is supported
- `DeleteObject` is supported
- the request body is buffered in memory so it can be replayed, bounded by `listener.replay_body_max_bytes` per request and `listener.replay_body_aggregate_max_bytes` across the process
- if any destination fails, the request fails overall
- upstream HTTP failures preserve the primary upstream error response when available
- transport or replay failures return a proxy-generated failure
- fan-out is not transactional; a destination that succeeds before another destination fails is not rolled back
- oversized replay attempts fail with `413 EntityTooLarge`; aggregate replay-budget exhaustion fails with `503 SlowDown`
- at most four destination attempts run concurrently; additional destinations wait for a slot

For writes matched by multiple routes through `on_match = "continue"`, every matched route must also succeed. A later route failure is returned as failure rather than hiding behind an earlier success.

## Outbound Signing

The proxy terminates and rebuilds requests before forwarding them.

Outbound S3 requests are signed with the destination backend credentials, not the inbound client credentials.

For outbound SigV4, the proxy uses `UNSIGNED-PAYLOAD`, and `Content-Length` must be set before signing.

### Upstream Redirects

Upstream `301`, `302`, `303`, `307`, and `308` responses are rejected for every
operation, including object reads, uploads, deletes and listings. The proxy
does not follow the redirect or forward its `Location` or response body. The
failure uses HTTP `502` with S3 `InternalError` through ordinary proxy error
handling. Configured `ordered_failover` may instead succeed at the next
configured destination; fan-out preserves an available primary non-redirect
HTTP error response under its normal rules.

Configure the correct backend endpoint and region. Redirect-based region
discovery is unsupported. Redirect URL credentials, object paths and query
values are excluded from errors and logs. Conditional `304 Not Modified` and
ordinary upstream HTTP error responses are unaffected.

## Error Behavior

Representative error rules:

- unsupported operations return S3-compatible `NotImplemented`
- route misses return standard S3-compatible error responses
- upstream backend failures propagate as proxy-mediated S3 responses
- multi-destination write failures are surfaced as failures, not partial success
- multi-route write failures are surfaced as failures, not partial success

## Health Endpoints

Local probes on the main S3 listener return `200 OK` with body `ok` only for **GET** requests with the exact escaped path `/healthz` or `/readyz`, no `Authorization` header (even an empty one), no query string (including a bare `?`), and a host that is not a configured virtual bucket host. Ordinary localhost, IP, and base-host probes work even on virtual-host-only listeners. `/readyz` means the proxy process is serving requests; it does not poll configured backends or report destination health.

All other requests follow the normal S3 parsing, authentication, authorization, and operation handling. This includes HEAD, signed or presigned requests, query-bearing requests, encoded spellings such as `/%68ealthz`, and requests to configured virtual bucket hosts. These paths can therefore be object keys on virtual bucket hosts, or path-style bucket names for ListObjectsV2 and HEAD. Malformed probe-path queries (for example, `/healthz?prefix=%GG`) return `400 InvalidRequest`. Unsupported operations retain their normal S3 errors; there is no probe-specific 405 response.

The indistinguishable unsigned, exact, query-free base-host GET shape is reserved for probes. Virtual bucket recognition uses the listener's enabled virtual-host addressing and ordered `host_suffixes`, including case, port, and trailing-dot normalization. Unconfigured host aliases cannot be inferred to be bucket hosts and may receive the local probe response. Configure probe hosts accordingly. Restrict probe access at the network or reverse-proxy layer if the listener is public. This GET-only contract replaces the former method-agnostic behavior; signed requests are no longer probes.
