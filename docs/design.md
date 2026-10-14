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

The operation classifier also owns the v1 query contract. Supported object, bucket-head, and virtual bucket-list operations accept no non-auth query keys except an optional AWS SDK `x-id` whose value matches the classified operation name. `ListObjectsV2` accepts `list-type=2`, `continuation-token`, `delimiter`, `encoding-type`, `fetch-owner`, `max-keys`, `prefix`, `start-after`, and optional `x-id=ListObjectsV2`.

Before classification, `requestctx.FromRequest` validates RawQuery through the addressing-independent `requestquery.Parse` boundary. It uses `url.ParseQuery`, discards the entire partial result on any error, and returns only the fixed `ErrMalformed` sentinel without wrapping parser excerpts. HTTP maps this to `400 InvalidRequest` before authentication, body reads/replay, routing, or dispatch. Invalid percent escapes and raw semicolons cannot make selectors or listing filters disappear. Probe eligibility still runs first, but any nonempty query falls through and receives this validation, including malformed healthz/readyz queries.

Standalone SigV4 `Verify` and direct backend `Do` enforce the same boundary before signing normalization/body verification and before source-body preparation/signing/network respectively. `buildTargetURL` also validates independently so direct helper callers cannot forward a partial decode. Validation never rewrites the source RawQuery: header verification signs a clone, and presign verification removes the signature from a freshly parsed map on a clone, retaining the existing signing behavior. Encoded semicolons/percent/plus, spaces, Unicode, opaque tokens, and operation-specific duplicate rules are preserved. Well-formed unsupported selectors retain `501 NotImplemented` and existing authentication precedence.

Request-derived query audit: auth and handler query consumers reuse validated values; backend classification and listing transformation use the strict source map. The remaining `s3ops.IsMultipart` calls require a strictly validated request, established by `FromRequest` in HTTP and by `Do` in the executor; this predicate is not an ingress validator. `targetURL.Query()` reads only the URL rebuilt with `Values.Encode()` by the guarded builder, and `signatureFromPresignedURI` reads only SDK-generated presign output. Test signature/fixture helpers consume their controlled well-formed inputs. New direct request boundaries must validate before calling such helpers.

Inbound SigV4 presign query keys are accepted for authentication and stripped before outbound signing. Every other query-key combination is unsupported and must be rejected before route dispatch, including ACL, tagging, retention, legal-hold, torrent, versioning and version IDs, restore, select, response-header overrides, and multipart query variants.

CopyObject and multipart-related operations are explicitly unsupported in v1 and
return an S3-compatible `501 NotImplemented` error. The shared `s3ops` boundary
detects `X-Amz-Copy-Source` on PUT and `X-Amz-Multipart-Upload-Id` on any S3 request
by case-insensitive header-name presence, regardless of values. Empty,
whitespace-only, repeated (including empty-first), and noncanonical direct-caller
header maps all count, including keys with nil or empty value slices. Ordinary
uploads must omit these selectors entirely; an empty value is not an opt-out.

The HTTP handler records classification and multipart detection before
authentication can filter unsigned headers, retaining the observed unsupported
state afterward. Rejection still follows authentication for supplied signatures:
invalid signatures keep their authentication error, and explicit payload hashes
are verified in the existing signature-then-body order. Strict query and streaming
payload checks keep their earlier precedence. Unsupported selectors are rejected
before routing or dispatch-related replay; the direct backend executor uses the
same detection before source-body preparation, `GetBody`, signing, or network I/O.

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

Malformed header or presigned `X-Amz-Date` values and invalid presigned
`X-Amz-Expires` values return fixed `AuthError` causes: `invalid X-Amz-Date` and
`invalid X-Amz-Expires`. The authentication parser discards value-bearing time and
integer parse errors, including overflow inputs, rather than wrapping them or
redacting only the top-level message. Returned errors and their unwrap chains
therefore retain useful `errors.Is` categories without retaining the rejected
values. The handler logs these categories in `auth failed` records and returns
the existing `403 AccessDenied` XML; request completion still records request ID,
method, status, response bytes, and duration.

Timestamp parsing, signature canonicalization, and validation order are preserved.
Header timestamps allow an inclusive 15-minute skew in either direction.
Presigned expiry remains an integer from zero through 604800 seconds (seven days),
with signing timestamps at most 15 minutes in the future. A request is expired
only when verification time is strictly after signing time plus expiry. Zero
expiry thus remains valid at the signing instant, and before it within the
future-skew allowance, but expires even one nanosecond afterward. Invalid date or
expiry inputs fail before signature/body verification, routing, or backend I/O;
valid signatures still precede explicit payload-hash verification.

### S3 Signature Canonicalization

Inbound header verification, presigned verification, and outbound signing use `DisableURIPathEscaping = true` with the AWS SDK v2 signer. The pinned `aws-sdk-go-v2 v1.43.0` signer obtains `URL.EscapedPath()` (or `URL.Opaque` when present), then applies an additional `EscapePath` by default. Its `SignerOptions` explicitly identifies S3 as requiring that additional escaping to be disabled. Both header signing and presigning share this canonical-request construction.

The canonical URI retains the escaped request path, including percent escapes, UTF-8 bytes encoded as escapes, escaped separators, repeated slashes, and dot segments. It is neither decoded nor path-normalized during signing. This is a compatibility correction for real S3 clients and backends; custom standalone SDK signers must enable the S3 option rather than reproduce the former generic double escaping. Tampering with the signed path or signature still fails authentication.

Payload verification and authenticated-header filtering are unchanged: verify the signature before reading an explicitly hashed body, check that hash against the bytes, and retain only authenticated control headers. Outbound requests use backend credentials and `UNSIGNED-PAYLOAD`, with upload `Content-Length` set explicitly before signing; inbound presign credentials are stripped from the outbound query.

Signing regressions use S3-configured SDK client fixtures and an SDK-independent test-only HMAC/SHA-256 calculation in `internal/testutil/s3signature`. Its canonical request uses the captured HTTP escaped path and an explicitly expected signed-header list. The calculation is anchored to the AWS S3 [header-signing GetObject example](https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html) (`20130524T000000Z`, `/test.txt`, `Range: bytes=0-9`, expected signature `f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41`), independently reproduced with Python's `hashlib`/`hmac`. This helper is a test oracle for these fixtures, not a general authentication implementation. Local HTTP tests run signed PUT and presigned GET through the real authenticator, authorizer, router, rewrite engine, dispatcher, and backend client, with the upstream checking independent signatures and stored object identity/bytes.

### Client Authorization

After successful authentication, the client can be authorized by:

- allowed routes
- optionally allowed operations
- visible virtual buckets for `ListBuckets`

Inbound client credentials and destination backend credentials are fully separate.

### Deferred Auth Features

The following are deferred to a later phase because v1 keeps credential management static and configuration-owned:

- credential generation
- credential rotation
- credential revocation
- admin API
- DB-backed client store

Residual risk: static credentials must be rotated and revoked outside the proxy until a DB-backed control plane exists. Deployments that require self-service credential lifecycle should not expose v1 as the credential authority.

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

`hash` computes SHA-256 over the original inbound bucket, a literal `/`, and the
original raw key, before any rewrite. The first four digest bytes are decoded as
an unsigned big-endian `uint32`; both that word and the positive destination
count are widened to `uint64` before modulo. The remainder is below the native
`int` destination count, so converting only the result to `int` is safe. This
avoids signed overflow and the unnegatable signed minimum, without narrowing a
64-bit destination count. The hash helper returns zero for counts at or below
one; the resolver skips routes with no available destinations and directly
selects a sole destination.

The index addresses destinations in configured order, not target-registry order.
No unescaping, escape-case folding, path normalization, or rewritten bucket/key
enters the hash. All previous 64-bit mappings are preserved. Some mappings on
older 386 and 32-bit ARM builds intentionally change because those builds folded
high-bit words through signed arithmetic. For three destinations, `bucket/key3`
(`ca7c2b99`) selects index 0, and `tenant-2/` (`f72858a6`) selects index 1 on every
corrected architecture; old 32-bit builds selected 1 and 0 respectively.

Mixed old-32-bit/corrected deployments can still disagree until rollout is
consistent. Keep the algorithm and ordered destination configuration consistent
across serving instances. This is read selection, not write sharding or data
migration: ordinary writes still select the first destination, and configured
`dispatch = "all"` still fans writes out. No stored objects or replica contents
are changed by the correction.

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

Prepending treats the incoming key as opaque bytes. One optional trailing slash
on the configured prefix supplies the join separator; no leading or repeated
slash in the key is removed. Thus `assets/` plus `foo`, `/foo`, and `//foo`
produces three distinct keys: `assets/foo`, `assets//foo`, and `assets///foo`.
Outbound URL construction preserves these slashes, including virtual-hosted
targets with endpoint base paths.

After authorization, all matched rewrites are checked before dispatch or
dispatch-related body replay. Object operations require a nonempty resulting
key, bucket operations require an empty key, and the resulting bucket must be
nonempty and contain no path or URL delimiters. Stripping or template output
that violates this shape returns `400 InvalidRequest` with S3 XML (a HEAD
response has no body on the wire). Template execution failures remain internal
errors. No matched route is dispatched when a rewrite fails preflight.

Object `key_template` execution failures return the fixed value-free error
`key_template execution failed`. The rewrite engine discards the `text/template`
execution error entirely, so error formatting and unwrap chains cannot retain
template names, failing action literals, capture names, or environment-derived
template contents. The HTTP `rewrite failed` record retains the public route
name and request ID; the client receives generic `500 InternalError` XML.

Successful compilation does not establish that a template can execute for every
request. Use the route and request ID to locate the rule, then inspect the parser's
named captures, capitalized template fields, and action argument types or indices
privately. Reproduce with synthetic matching requests. Offline validation checks
syntax and the restricted listing contract; runtime capture availability and
action evaluation still depend on request data.

The shared operation vocabulary defines outbound method/bucket/key validation.
The backend executor repeats validation before reading or replaying a body,
signing, or making an upstream call, and checks query/header classification
against the declared operation. Unsupported operations and malformed direct
executor requests return errors. `ListBuckets` remains local to the proxy.

`HeadBucket` applies only the bucket override, ignoring all object key rules and
addressing the backend bucket root. It checks bucket availability/permission,
not the existence of a virtual prefix. `ListObjectsV2` also uses the bucket root,
with a separate namespace mapping as described below.

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
- optional request timeout

### Static Backend Session Credentials

`credential "static"` accepts an optional `session_token` alongside the required
access and secret keys. HCL literals and native `env()` results pass unchanged
from `rawCredential` to `StaticCredential` and the selected target's
`aws.Credentials.SessionToken`. An omitted or empty token retains long-lived key
behavior. This is a backend credential field; the inbound `Client` schema and
authentication lifecycle are separate.

The executor owns a startup snapshot of its target registry. Credentials are
string-valued structs copied with each target, so caller mutation or later
environment changes cannot alter the selected target's credentials or another
destination. There is no credential provider chain, STS call, issuance, automatic
refresh, or expiry tracking. Operators replace the complete externally issued
credential set and restart before expiry.

Header/query filtering removes inbound `X-Amz-Security-Token` case-insensitively
before signing a fresh outbound request. Only the selected target's configured
token can be emitted. The existing AWS SDK v2 signer places it in the outbound
security-token header and includes that header in SigV4 `SignedHeaders`.
`UNSIGNED-PAYLOAD`, explicitly signed upload `Content-Length`, and S3 canonical
path handling retain their existing contracts. No mutable credential state is
shared between concurrent fan-out attempts or tokenless destinations.

Credential fields stay outside topology and runtime log payloads; existing HCL
diagnostic sanitization also applies to environment-derived token type errors.
Local config/app HTTP workflows and direct executor captures verify token
isolation and snapshots. The existing independent S3 signature oracle uses an
explicit expected signed-header list and rejects token mutation or removal.

## Forwarding Behavior

### Upstream Redirect Boundary

The shared S3 executor rejects upstream `301`, `302`, `303`, `307`, and `308`
responses, even without a Location header. A redirect cannot change the
validated operation, path, host, or signed body. Operators must configure the
correct backend endpoint and region; there is no redirect-based region
discovery or signing of redirect destinations.

`NewClient` copies the supplied `http.Client`, retaining its timeout and cookie
jar and delegating to its existing transport (or the default transport). A
transport wrapper rejects these statuses before net/http parses Location,
creates a replay body, or invokes any caller redirect callback. Checking only
`CheckRedirect` is insufficient: malformed Location parsing occurs before that
callback and can embed redirect secrets in errors. The copied client also uses
a refusal callback; the caller-owned client and transport are not modified.

Rejection closes the upstream response body exactly once without reading it,
discards its headers/body and any Close error, and returns the fixed
`upstream redirect rejected` cause. The executor's ordinary error path releases
target cancellation. Redirect credentials, paths, queries and response bytes
are not included in errors or logs. Request replay remains owned by the caller:
the HTTP handler releases its reservation on exit, including fan-out failures;
direct executor callers release their source via `replaybody.Release` as usual.

The existing proxy error path returns HTTP `502` with S3 `InternalError` and no
redirect Location. Configured ordered failover can try another configured
destination after this backend request error. Fan-out retains its existing
primary non-redirect HTTP error precedence. Ordinary success, conditional `304`,
and upstream HTTP error responses retain their streaming/forwarding contracts.

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
- upstream HTTP failures preserve the primary upstream error response when available
- transport or replay failures return a proxy-generated failure
- log per-destination result details

For multi-route writes collected by `on_match = "continue"`, every matched route must also succeed. A failure from any matched route makes the overall request fail, even if an earlier route returned success.

Reads do not fan out in v1.

### Replay Storage And Budget

One app-scoped budget is shared by payload-hash verification, multi-route writes,
fan-out, and unknown-length outbound uploads. The per-request limit defaults to
32 MiB (`listener.replay_body_max_bytes`); the aggregate payload limit defaults to
256 MiB (`listener.replay_body_aggregate_max_bytes`). Per-request overflow returns
`413 EntityTooLarge`; aggregate exhaustion returns `503 SlowDown` immediately.

Known lengths reserve and retain one exact-capacity payload allocation. Unknown
lengths coalesce source reads into 32 KiB chunks, copying only full chunks and the
final partial chunk into exact-capacity retained slices. For a payload of N bytes,
there are exactly ceil(N / 32768) retained chunks (zero for an empty body), with
total payload capacity N. Every positive source read is charged immediately,
including partial chunks awaiting completion; short reads do not multiply retained
allocations or slice descriptors. Chunk metadata grows with chunk count, with
ordinary Go slice-capacity growth, rather than with source read count.

The allowance outside payload accounting is exactly one 32,768-byte scratch
buffer per active unknown-length buffering operation. There is no additional
final-chunk capacity allowance: even the final partial chunk has capacity equal
to its length. Slice descriptors (24 bytes each on 64-bit Go, including unused
descriptor capacity), request/replay bookkeeping, and Go allocator rounding are
additional overhead; the payload budget is not a process RSS limit. Scratch is
transient and is not retained by installed replay readers.

EOF with data is accepted after accounting and cancellation checks. Read errors,
overflow, aggregate exhaustion, cancellation, and source-close errors release the
operation's reservations. Unknown-length readers returning `(0, nil)` 100 times
consecutively fail with `io.ErrNoProgress`; positive progress resets that count.
Cancellation closes a blocked source through the existing close-once path.

Successful replay remains caller-owned. Independent `GetBody` readers share
immutable payload storage without owning the reservation. `Reset` rewinds by
replacing the current body with such a reader; closing it does not end ownership.
Closing the initially installed owning body, cancellation, or `replaybody.Release`
returns accounting once. `Release` also ends ownership after `Reset` and clears
the request's `Body` and `GetBody` hooks.

Each nonempty payload initializes a release-notification channel before starting
its cancellation watcher. The watcher selects between context cancellation and
that notification. Once-only release returns the budget and closes the channel
without waiting for the watcher, so immediate cancellation and explicit release
can race without a registration gap or recursive `sync.Once` wait. Explicit
release does not cancel the caller's context. The watcher can exit promptly even
when a direct caller keeps that context alive; ordinary net/http request contexts
already cancel when the handler returns.

Release never clears or mutates the immutable payload bytes. Already-acquired
readers and saved `GetBody` closures remain byte-valid after release or
cancellation. Retaining these references, including a closed owning body, retains
their storage intentionally. Once callers discard them, storage can be collected
without waiting for context cancellation; an otherwise unnecessary watcher no
longer extends its lifetime. Callers must still observe this replay lifetime
discipline when interpreting the payload budget.

## Listing Semantics

### `ListBuckets`

`ListBuckets` is supported as a proxy-defined virtual view.

It does not aggregate actual upstream bucket listings.

Instead, the proxy returns buckets explicitly exposed in config and visible to the authenticated client.

This keeps the behavior:

- predictable
- secure
- independent of backend-specific bucket visibility

The `internal/listbuckets` service copies visible names and the supplied startup
creation date into a private, lexicographically ordered snapshot at construction.
Each discovery result is a caller-owned slice; configuration and principal policy
slices retain their original contents and order. Nil principals and policies
containing `*` receive a copy of the whole catalog. A nonnil principal with a nil
or empty visibility policy receives no buckets. Explicit policies use a call-local
name set to filter the ordered snapshot, with expected O(B + V) discovery work for
B catalog buckets and V policy entries. Ordering is paid once at startup.

Visibility depends on `visible_buckets`, independently of route permissions.
Bucket `route` references serve configuration referential integrity; the HTTP
handler separately checks the client's `ListBuckets` operation permission.

### `ListObjectsV2`

`ListObjectsV2` is supported only against one effective backend in v1.

Supported reversible mappings are bucket-only rewrites, prepend prefixes, and
templates composed of literal text, `.Bucket`, and `.Captures.name`, followed by
exactly one terminal `.Key`. The template AST is validated; functions, pipelines,
variables, control flow, additional template definitions/calls, nonterminal/repeated keys, and
key-independent templates are rejected on listing routes. `.Bucket` is the
rewritten bucket, consistent with object rewriting. Prepend may be combined
with a prefix template: the effective prefix is the template prefix followed by
the prepend prefix (including its join slash). Capture values must be present.

All `strip_key_prefix` and `strip_path_prefix` combinations are unsupported for
listing and rejected at startup, including apparently redundant strips. The
path-style parser already removes the bucket segment; public examples therefore
need no strip rule. General object-only templates and strip rules remain usable
on routes without `ListObjectsV2`. Runtime checks also reject unsafe mappings
before dispatch with `400 InvalidRequest`, including missing captures or invalid
percent escapes that cannot be diagnosed at startup.

The reusable `internal/namespace` mapping keeps raw HTTP paths separate from S3
key values. Configured/template prefixes use raw path notation and must contain
well-formed percent escapes for listings. `%2F` in a raw prefix represents a key
slash, `%252F` represents the literal key text `%2F`, and `%25` represents `%`.
The combined prefix is path-unescaped exactly once. Client query values have
already been query-decoded and are never path-unescaped. They are concatenated
with the decoded namespace prefix for backend `prefix` and optional `start-after`.
An absent or empty visible prefix still scopes the backend request. URL escaping
is applied by normal query serialization before outbound signing.

Successful XML is parsed strictly and translated before any response is committed:
`Name` becomes the visible bucket; `Prefix`, `StartAfter`, `Contents.Key`, and
`CommonPrefixes.Prefix` become visible keys. `encoding-type=url` means percent
decoding these XML key fields once (`+` or `%20` for space, `%2B` for literal
plus), then percent encoding the translated values with `%20` for spaces.
This accepts the form-style encoding used by MinIO and SeaweedFS as well as
percent-only encoders. Delimiter is validated and retained in
the requested encoding. Without URL encoding, key text is plain XML character
data. Ordinary XML entities are handled by the XML parser/encoder. A listed key
must be requested with normal S3 path escaping for GetObject; its XML spelling
is not an opaque HTTP RawPath. Leading/repeated slashes, percent text, Unicode,
and dot segments remain key data without path cleaning.

The upstream Name, Prefix, encoding and optional StartAfter/Delimiter/token
echoes must match the request. For compatibility with SeaweedFS, known query
echoes (Prefix/StartAfter/Delimiter) may be either encoded or exactly equal to
the decoded request value; output always uses the requested encoding. This
exception does not apply to returned object keys/common prefixes. Every returned
key/common prefix must be within
the requested backend prefix, not merely the broader namespace. A sibling or an
object exactly equal to the hidden namespace prefix fails the entire response;
the latter would have an empty visible object key. Entries are never silently
filtered because that would invalidate counts and pagination. Standard V2 XML
fields (including owner, checksum, and restore metadata) are preserved; unknown
elements, unexpected nesting/namespaces/attributes, duplicate singleton fields,
invalid encoding, malformed XML, DTDs, and multiple roots fail closed.

Leaf character data accumulates in a parser-local growable buffer and becomes
an immutable element string once, at the leaf's closing tag. Valid leaves cannot
nest, so a single mutable accumulator suffices and is reset between leaves;
mutable buffers are never stored in the translated element tree or shared across
requests. Plain text, entities, comments, and CDATA retain their XML semantics.
Accumulation takes amortized linear copying/allocation in decoded text size even
when comments or CDATA split a leaf into many tokens. Element/byte/deadline limits
alone would not bound the cumulative allocation of repeated string concatenation.

Transformation limits are fixed: 8 MiB input and output, 50,000 XML elements,
four element levels, and five seconds for reading/parsing/encoding. Earlier
request cancellation or target timeout wins. A context callback closes the
upstream HTTP body to interrupt a blocked read; all exits close it exactly once.
Limits apply per response, not as an aggregate process memory budget. Invalid or
oversized upstream listings return HTTP `502` with S3 `InternalError` without exposing partial XML.
The transformed response has fresh Content-Type/Content-Length and no stale
framing, content encoding, ETag, digest, checksum, or range headers. Upstream
HTTP error bodies remain ordinary errors; GetObject continues to stream.

If a route has multiple destinations, `read_preference` selects the single
backend used for the list call. `ContinuationToken` and `NextContinuationToken`
are opaque strings: neither URL-decoded, namespace-stripped, nor interpreted as
keys (even if they contain a backend prefix). Their query values are normally
escaped on transmission. `random` may change backend on every page;
`ordered_failover` may change backend on an error, including a failed listing
transformation. Tokens may then be invalid or produce inconsistent pages.
`first`, or `hash` with the same ordered destination configuration and corrected
algorithm on every serving instance, keeps bucket listing selection stable.
`hash` uses the original bucket and empty key, ignoring list filters and tokens,
so later object reads can select another replica. Operators need equivalent
replicas for that workflow. Replacing old 32-bit hash instances or changing
destinations can change the backend between pages. Complete pagination with
stable routing before that change, or restart affected listings after rollout.
There is no proxy-managed backend affinity encoded in tokens.

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

### Interrupted Response Transfers

All forwarded responses use one streaming writer, including upstream error XML
and the bounded XML body produced by listing transformation. Object bodies
remain streamed with `io.Copy`; this path does not buffer whole downloads.
After headers are committed, a copy failure is logged with the existing URL
sanitizer and the handler panics with `http.ErrAbortHandler`. Deferred body
closure runs before net/http aborts the HTTP/1 connection or resets the HTTP/2
stream. No replacement XML is appended to the partial response. This also
prevents net/http from turning a small buffered partial body into a normally
completed response with an inferred Content-Length.

Body closure is deferred before forwarding headers, so read/write panics also
release the body. A Close-only error after a successful copy is logged as
`response cleanup failed` and does not abort delivery. Copy and Close failures
are logged separately when both occur. The deferred `request complete` record
still reports the original status, bytes accepted by the response writer, and
duration during abort unwinding; it is a lifecycle record, not proof of client
receipt. Middleware must propagate `http.ErrAbortHandler` to net/http.

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
> The `env("VAR")` call is a native HCL function supported in string attributes.
> It returns the environment value as a literal cty string during decoding;
> `${...}` and `%{...}` in that value are never evaluated as HCL templates.
> Unset variables return an empty string, followed by ordinary field validation.
> Parsing uses the original bytes, preserving filenames, line numbers, and
> columns in diagnostics. Comments and literal `env(...)` text are untouched;
> function-call whitespace is handled by the HCL parser.
> All environment results are sensitive for diagnostic purposes, regardless of
> the destination field. After decoding, compiler/validator errors omit all
> attribute values (including literal ones), so derived reference suffixes,
> escaped strings, and parser excerpts cannot bypass a string-redaction rule.
> URL errors are reduced to safe categories, regex errors to their syntax code,
> durations to the expected syntax, and Go-template errors to safe categories;
> raw parser errors are never wrapped into returned configuration errors.
> Compilation retains original attribute-expression ranges and block/field
> identity. Validation retains block/field identity and the violated rule.
> At the HCL boundary, diagnostic subjects are associated with their original
> attributes. If an attribute expression contains an `env()` call anywhere in
> its syntax tree (including interpolation, conditionals, nested calls, and
> comprehensions), value-bearing diagnostic details are withheld. HCL's safe
> summary/category and original subject range remain, with block/field context;
> duplicate-object-key diagnostics also withhold the evaluated key for literal
> expressions, retaining a fixed uniqueness cause. Other ordinary public HCL
> diagnostics retain their details. This uses source
> provenance, not matching environment contents. No raw diagnostic evaluation
> context or parser cause is returned. Filenames and source-literal block labels
> are public diagnostic metadata. Runtime values are not rewritten or redacted.
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
  timeout          = "5s"
  credentials      = "primary"
}

target "s3" "replica" {
  endpoint         = "https://minio-b.internal"
  region           = "us-east-1"
  force_path_style = true
  timeout          = "5s"
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

Listener addresses are checked at the shared `config.Validate` boundary used by
`Load`, `LoadFile`, app construction, and offline CLI inspection. `net.SplitHostPort`
checks TCP separator/bracket structure; URL-form addresses are rejected explicitly
(including a URL without a port that the splitter could mistake for a service).
Parser errors are discarded, not wrapped: diagnostics contain only the public
listener label, `address` field, and a fixed structural or numeric-range cause.
Malformed addresses therefore fail before topology/success output or startup
logging, intentionally earlier than the former `net.Listen` failure.

The numeric port check is local and bounded against overflow. Empty ports mean
zero; optional signs and leading decimal zeros are preserved, including negative
zero and Go's bare-sign zero behavior. Numeric values must be in `0`–`65535`.
The check preserves Go's `net.parsePort` early numeric-overflow boundary: a
nondigit allows service lookup only if reached before the parser's cutoff or
uint32 addition overflow. Thus `1073741824service` remains a runtime service input but
`10737418240service` and `4294967296service` fail numeric validation locally.
Go's uint32 multiplication can wrap before a suffix; service classification keeps
that behavior (`4294967300service`), while a separate sticky range flag rejects
all purely numeric values above 65535 even if Go's accumulator later wraps.
Nonnumeric service names remain runtime inputs. Wildcard hosts, IPv4, hostnames,
and bracketed IPv6 with zones keep Go's structural contract; the validator adds
no DNS naming policy. No resolution, service lookup, interface inspection, or
binding occurs here. Service existence, hostname/zone resolution, address ownership,
permissions, and port availability remain runtime checks. Valid occupied or
unavailable addresses must continue to pass `validate` and `routes` offline.

Target endpoints are checked at the same shared `config.Validate` boundary
against the components consumed by `buildTargetURL`: an absolute, hierarchical
HTTP(S) URL with a nonempty hostname, optional decimal port in `0`–`65535`, and
optional escaped base path. Userinfo, query (including `ForceQuery` for a bare
`?`), fragment, and opaque forms are rejected rather than silently discarded.
Both parsed fragment fields and the original `Endpoint` string are checked:
Go's parsed URL does not retain an empty `#` delimiter. Literal `%3F`/`%23` path
escapes remain valid. Credentials belong to the separate referenced static
credential block, never endpoint userinfo.

Reparsing `EndpointURL.String()` applies Go's URL syntax rules to direct runtime
callers as well as loaded HCL. URL parser causes and numeric-overflow errors are
discarded; diagnostics retain only public target labels, the endpoint field,
and fixed useful categories. Validation does not rewrite either endpoint form.
DNS/IPv4 targets keep both addressing modes. Bracketed IPv6 targets, including
zones and IPv4-mapped addresses, require `force_path_style = true` because the
virtual-hosted builder cannot prefix a bucket to that authority. Optional/empty
ports, leading decimal zeros, and escaped base paths retain their behavior.
No additional DNS naming policy, resolution, interface checks, bind attempts,
or backend calls are introduced. These configuration failures precede CLI
success/topology output and app startup logs. Migration removes formerly ignored
components, corrects malformed hosts/ports, and selects path-style for IPv6.

## CLI Design

The service is a single binary named `s3proxy`.

Recommended commands:

- `s3proxy serve --config /etc/s3proxy/config.hcl`
- `s3proxy validate --config /etc/s3proxy/config.hcl`
- `s3proxy routes --config /etc/s3proxy/config.hcl`
- `s3proxy print-example-config`
- `s3proxy version`

### Delivered Static Starter

`print-example-config` is implemented in `cmd/s3proxy/example.go`. Its no-argument
command writes a version-owned HCL constant with a trailing newline, without
loading configuration, reading environment values, creating an app/runtime, writing
files, or performing network/bind operations. Unknown flags and positional arguments
fail; write errors and short writes propagate as command errors. Shell redirection
owns file creation and any partial output on failure.

The starter uses native loopback `127.0.0.1:8080`, path-style addressing,
`sigv4_static`, and separate literal client/backend credential `env()` references.
One exact-bucket route exposes `images` as `images-store` on target `primary` in
`us-east-1`, retaining object keys and a root list/head namespace. Explicit client
permissions cover only that route/bucket and the six routed object/list/head
operations, plus local virtual `ListBuckets` discovery. Backend provisioning and
credential choice remain operator responsibilities.

Explicit replay bounds are 32 MiB per request and 256 MiB aggregate; read-header,
read, write and idle deadlines are 10 seconds, 2 minutes, 5 minutes and 60 seconds,
with a 2-minute complete upstream timeout. These settings favor bounded modest
transfers and must be sized for deployment concurrency, memory and link speeds;
replay bounds are not universal object-size limits. The native quickstart documents
print → variable setup → validate → routes → serve. Docker instructions preserve
the serve-specific entrypoint, override it for offline commands, require `:8080`
inside the container, and pair restricted host publishing with external TLS.
The manual auth-none quickstart remains a distinct trusted-testing workflow.

### Offline Route Topology

`routes --config PATH` / `routes -c PATH` loads and fully validates configuration
through `config.LoadFile`. It does not construct the application, initialize
runtime logging, bind listeners, or call backends. Missing config and positional
arguments are errors. Invalid configuration returns the existing safe diagnostics
and a nonzero exit status before any report is written; output errors also fail
the command (an output device may already have accepted a partial write).

The JSON boundary is an explicit private DTO in `cmd/s3proxy/routes.go`, never
serialization of `config.Runtime`, `Route`, `Parser`, or `S3Target`. The top-level
`routes` array is in declaration order (`[]` when there are no routes). Each entry
contains only one-based `ordinal`, route `label`, `parser` with declaration `label`
and validated `kind`, ordered `operations`, ordered destination declaration labels
in `destinations`, and validated `dispatch`, `on_match`, and `read_preference`.
References are looked up in the validated declaration maps; even an
environment-derived qualified reference prefix is never output. No maps are
serialized, so repeated runs with the same evaluated topology are byte-identical.
Output is two-space-indented JSON with a trailing newline.

Route operation filters are exact, case-sensitive lists, not inferred capabilities:
empty, omitted, duplicate, unknown, and wildcard entries fail validation. The
resolver matches only explicitly listed operations; an empty list would match
nothing, not everything. `dispatch` and `on_match` are required; omitted or empty
`read_preference` resolves to `first`. The report includes every configured route,
even if an earlier route or authorization policy might prevent a request reaching
it. It does not evaluate matches, rewrites, authorization, health, or select an
effective read destination. `ListBuckets` remains locally served virtual listing
even when included in a route's configured operations.

Source-literal declaration labels are public metadata. All other emitted strings
are finite-vocabulary fields checked by full configuration validation, including
when sourced from `env()`. Arbitrary attribute values are excluded: credentials,
endpoints, regions, listener addressing, parser matches/regexes, rewrites/templates,
visible bucket values, environment names/values, and raw reference expressions.
Future configuration fields cannot enter this DTO automatically. Existing
diagnostic confidentiality rules still apply on failure, including public config
filenames and source labels. The operator schema/example is documented in
`website/docs/operations.md#offline-route-topology`.

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
- logs, health, readiness

Local process probes require GET, exact `URL.EscapedPath()` `/healthz` or `/readyz`, no Authorization header present (including empty values), empty RawQuery with ForceQuery false, and no virtual bucket host recognized by the listener's configured addressing. The handler and request parser share virtual bucket recognition, preserving host case/port/trailing-dot normalization, dot boundaries, nonempty bucket names, suffix order, and virtual-host enablement. Probe eligibility is checked before address parsing so localhost/IP/base-host probes still work on virtual-host-only listeners.

Only this indistinguishable unsigned, exact, query-free base-host GET shape is reserved. Every other shape follows the normal S3 pipeline, including HEAD, signed/presigned requests, query strings (even bare `?`), encoded spellings, and configured virtual bucket hosts; no probe-specific 405 is introduced. Thus healthz/readyz object keys and path-style ListObjectsV2/HEAD bucket operations remain available. Unconfigured host aliases cannot be inferred to be virtual bucket hosts. This intentionally replaces method-agnostic probe interception and signed probe success.

Eligible probes return `200 OK` with body `ok` and retain request IDs and completion logging. `/readyz` reports startup readiness only: if the listener is serving requests, an eligible probe succeeds; it does not poll configured backends or infer destination health.

### Milestone 2

- multi-destination fan-out for:
  - `PutObject`
  - `DeleteObject`
- stronger failure handling
- more logging detail
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
- credential generation is deferred to a later DB-backed phase, with credential lifecycle handled outside the proxy in v1
