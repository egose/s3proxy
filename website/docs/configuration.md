---
sidebar_position: 3
---

# Configuration

`s3proxy` uses labeled HCL blocks. Listener, credential, target, and parser blocks have a type and name label; auth, route, and bucket blocks have one name label. The core building blocks are:

- `listener "http" "public"`
- `auth "main"`
- `credential "static" "primary"`
- `target "s3" "primary"`
- `parser "path_prefix" "images"`
- `route "images_rw"`
- `bucket "images"`

## Mental Model

Think about the config in seven layers:

1. `listener` defines how the proxy accepts traffic.
2. `auth` defines whether clients are authenticated and what they can see.
3. `credential` defines backend credentials stored in config.
4. `target` defines the S3-compatible backend endpoints.
5. `parser` defines how requests are matched.
6. `route` combines parser, operation filters, destinations, and rewrite rules.
7. `bucket` defines the virtual buckets returned by `ListBuckets`.

## Example

```hcl
listener "http" "public" {
  address = ":8080"
  replay_body_max_bytes = 33554432
  replay_body_aggregate_max_bytes = 268435456

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
    access_key      = env("S3PROXY_CLIENT_CI_ACCESS_KEY")
    secret_key      = env("S3PROXY_CLIENT_CI_SECRET_KEY")
    allow_routes    = ["route.images_rw"]
    visible_buckets = ["images"]
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

route "images_rw" {
  parser          = "images"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject"]
  destinations    = ["primary"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    prepend_key_prefix = "assets/"
    bucket             = "images-store"
  }
}

bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
```

## Listener

The listener block configures the inbound HTTP server.

Important fields:

- `address`
- `max_header_bytes`
- `replay_body_max_bytes`
- `replay_body_aggregate_max_bytes`
- `addressing.path_style`
- `addressing.virtual_hosted`
- `addressing.host_suffixes`
- `timeouts.read`
- `timeouts.read_header`
- `timeouts.idle`
- `timeouts.write`

Notes:

- only one `listener` block is supported
- only `listener "http" ...` is supported in v1
- `address` and `addressing.path_style` are required
- enabling `virtual_hosted` requires at least one `host_suffix`
- `max_header_bytes` defaults to `1 MiB`; `0` also uses that default
- `timeouts.read` defaults to `30s`; omitting it or setting `0s` retains that default
- `timeouts.read_header`, `timeouts.idle`, and `timeouts.write` are disabled when omitted or set to `0s`
- `replay_body_max_bytes` caps per-request buffering when the proxy must replay a request body; `0` uses the default `32 MiB`
- `replay_body_aggregate_max_bytes` caps retained replay buffers across the process; `0` uses the default `256 MiB`
- replay-bound requests fail with `413 EntityTooLarge` when the per-request limit is exceeded and `503 SlowDown` when the aggregate budget is exhausted

Replay buffering is used for fan-out writes, writes matched by multiple routes, inbound SigV4 requests with a concrete payload hash, and outbound requests whose body length is unknown.

### Listener Address Validation

`address` must be a nonempty TCP `host:port` string, not a URL. Configuration
loading (including `validate`, `routes`, and `serve`) checks Go's host/port
separator and bracket structure and rejects numeric ports outside `0`–`65535`.
For example, `127.0.0.1`, `http://127.0.0.1:8080`, and `:65536` now fail during
configuration validation rather than later at startup. Errors identify the
listener label and `address` field without echoing the address or parser excerpts.

Accepted offline forms include:

- wildcard hosts: `:8080`, `0.0.0.0:8080`, `[::]:8080`
- IPv4 and hostnames: `127.0.0.1:8080`, `gateway.example.com:8080`
- bracketed IPv6, including zones: `[::1]:8080`, `[fe80::1%eth0]:8080`
- named service ports: `localhost:http`, `:https`
- ephemeral ports: `127.0.0.1:0` or an empty port such as `127.0.0.1:` or `:`

Numeric ports are decimal; leading zeros and an optional sign retain Go's
behavior (`:+8080`, `:00080`, and `:-0` are accepted). Go also treats a bare `+`
or `-` port as zero. Negative nonzero numeric ports are rejected. Nonnumeric
service names are left for runtime resolution; validation does not check that
the service exists. Go stops parsing sufficiently large numeric prefixes before
reaching any service suffix; those overflow forms (such as `:10737418240service`)
are also rejected offline.

This is an offline syntax gate, not a bind-availability check or a DNS naming
policy. Hostname/service resolution, IPv6 zone/interface existence, local address
ownership, bind permissions, and port availability are checked at runtime.
Syntactically valid unavailable addresses and occupied ports still pass
`validate` and `routes`; neither command binds a listener or contacts backends.

## Auth

Supported inbound auth modes:

- `none`
- `sigv4_static`

`none` skips inbound authentication and is only appropriate in trusted environments.

`sigv4_static` validates inbound S3 SigV4 signatures against configured `client` blocks.

Each client may define:

- `access_key`
- `secret_key`
- `allow_routes`
- `allow_ops`
- `visible_buckets`

`access_key` and `secret_key` are required. Policy list defaults are intentionally asymmetric:

- omitted or empty `allow_routes` denies every routed operation
- omitted or empty `allow_ops` allows every supported operation
- omitted or empty `visible_buckets` returns no buckets from virtual `ListBuckets`
- `"*"` allows every known value for that policy field

Non-`ListBuckets` requests must match both `allow_routes` and `allow_ops`. `ListBuckets` checks `allow_ops` and filters its response through `visible_buckets`; it does not resolve a route.

Example:

```hcl
auth "main" {
  mode = "sigv4_static"

  client "admin" {
    access_key      = env("S3PROXY_CLIENT_ADMIN_ACCESS_KEY")
    secret_key      = env("S3PROXY_CLIENT_ADMIN_SECRET_KEY")
    allow_routes    = ["*"]
    allow_ops       = ["*"]
    visible_buckets = ["*"]
  }
}
```

## Credentials

Backend credentials are defined separately from auth clients:

```hcl
credential "static" "primary" {
  access_key = env("S3PROXY_TARGET_PRIMARY_ACCESS_KEY")
  secret_key = env("S3PROXY_TARGET_PRIMARY_SECRET_KEY")
}
```

Only `credential "static" ...` is supported in v1.

## Targets

Targets define outbound S3-compatible backends:

```hcl
target "s3" "primary" {
  endpoint         = "https://minio-a.internal"
  region           = "us-east-1"
  force_path_style = true
  timeout          = "5s"
  credentials      = "primary"
}
```

Supported fields:

- `endpoint`
- `region`
- `force_path_style`
- `timeout`
- `credentials`

Only `target "s3" ...` is supported in v1.

`endpoint`, `region`, and `credentials` are required. The endpoint must be an absolute `http` or `https` URL, and the credential reference must exist. `force_path_style` defaults to `false`, which uses virtual-hosted outbound addressing. An omitted or zero `timeout` adds no whole-request target deadline; a positive value covers the complete upstream exchange, including streaming the response body. The shared upstream transport separately uses a 10-second dial timeout, 10-second TLS handshake timeout, and 30-second response-header timeout.

## Parsers

Supported parser kinds:

- `path_prefix`
- `bucket_exact`
- `bucket_regex`
- `host_suffix`

Examples:

```hcl
parser "path_prefix" "images" {
  prefix = "/images"
}

parser "bucket_exact" "logs" {
  bucket = "logs"
}

parser "bucket_regex" "tenant_logs" {
  pattern = "^tenant-(?P<tenant>[a-z0-9-]+)-logs$"
}

parser "host_suffix" "public_hosts" {
  suffix = "s3proxy.example.com"
}
```

Named regex captures from `bucket_regex` are available to `key_template` rewrites.

## Routes

Routes are evaluated in config order.

Each route combines:

- `parser`
- `operations`
- `destinations`
- `dispatch`
- `on_match`
- `read_preference`
- `rewrite`

Important route fields:

- `dispatch = "first"` sends the request to one destination
- `dispatch = "all"` fans writes out to all destinations
- `on_match = "stop"` stops route evaluation after this match
- `on_match = "continue"` keeps collecting matches

`parser`, `operations`, `destinations`, `dispatch`, and `on_match` are required. `read_preference` defaults to `first`.

`on_match = "continue"` is valid only for write-only routes. A `dispatch = "all"` route must contain `PutObject` or `DeleteObject`; it may also contain supported reads, which still use only one destination. For a write-only `dispatch = "all"` route, `read_preference` must be `first`.

For multi-destination writes and multi-route writes collected with `on_match = "continue"`, every selected destination or route must succeed. Upstream HTTP failures preserve the primary upstream error response when available; transport or replay failures return a proxy-generated failure.

Supported `read_preference` values:

- `first`
- `random`
- `hash`
- `ordered_failover`

`first` selects the first configured destination. `random` selects a destination for each resolution. `hash` deterministically selects from the original inbound bucket and key before rewrites. `ordered_failover` starts with the first configured destination and tries later destinations on errors returned while preparing, signing, or sending the upstream request and on upstream `5xx` responses. It does not fail over on upstream `4xx` responses such as `404`, `NoSuchKey`, or `NoSuchBucket`.

## Rewrites

Supported rewrite fields:

- `strip_path_prefix`
- `strip_key_prefix`
- `prepend_key_prefix`
- `bucket`
- `key_template`

Example:

```hcl
rewrite {
  prepend_key_prefix = "assets/"
  bucket             = "images-store"
  key_template       = "{{ .Captures.tenant }}/{{ .Key }}"
}
```

Template data uses the names `Bucket`, `Key`, and `Captures`.

`HeadBucket` applies only `bucket` and targets the backend bucket root. It does
not check whether a virtual prefix exists.

`ListObjectsV2` supports bucket-only rewrites, prepend prefixes, and prefix-only
templates made of literals, `.Bucket`, and `.Captures.name`, followed by exactly
one terminal `{{ .Key }}`. These prefixes may be combined; `.Bucket` is the
rewritten bucket. Template functions, pipelines, control flow, additional definitions/calls,
variables, repeated/nonterminal keys, and all strip-key/strip-path combinations
are rejected at startup on listing routes. Path-style parsing already removes
the visible bucket segment, so redundant stripping should be removed.

Configured prefixes use raw path notation with valid percent escapes: `%25`
means a literal percent, `%2F` a slash, and `%252F` literal `%2F` key text. Missing
captures or invalid runtime prefix escapes return `400 InvalidRequest` before
dispatch. Object-only routes can still use general templates and strip rules.
Query/XML keys use S3 key values instead of raw paths; list filters and results
are translated into the visible namespace. See [the listing contract](api-reference.md#listobjectsv2)
for encoding, bounds, and pagination limitations.

## Virtual Buckets

`ListBuckets` returns buckets you define explicitly:

```hcl
bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
```

`visible_name` is what the client sees in virtual `ListBuckets` responses. `route` must reference an existing route, but it is used only for configuration referential integrity. It does not create a request route or couple bucket visibility to route authorization.

Although `ListBuckets` is accepted as an operation name during route validation, it is handled before route resolution. Adding it to a route has no routing effect.

## Environment Variables

Use `env("VAR")` anywhere a string is allowed. This native HCL function returns
the environment value as a literal string during evaluation. `${...}`,
`%{...}`, quotes, newlines, backslashes, and Unicode in the value are data;
they are never parsed as HCL source or templates.

Ordinary function-call whitespace works, including `env ( "VAR" )`. Comments
and literal strings containing `env(...)` text are not evaluated. HCL source
can explicitly interpolate a call, such as `"prefix-${env("VAR")}"`; the
returned value is still literal and is not evaluated again. Field-specific
processing still applies, such as URL parsing for `endpoint` and Go template
compilation for `key_template`.

An unset variable returns an empty string. There is no separate missing-variable diagnostic; required-field validation may reject the resulting value, while optional string fields may remain empty.

The loader parses the original file bytes, so parse/decode diagnostics retain
the original filename, line, and column, including leading blank lines.

Every `env()` result is treated as sensitive in diagnostics, not just credential
values. HCL errors in expressions that call `env()` retain their category,
block/field identity, and original location, but withhold value-bearing details.
This includes explicit interpolation, nested calls, and collection expressions.
Duplicate-object-key errors also withhold the evaluated key for literal
expressions, while retaining the uniqueness cause and block/field location.
Other ordinary HCL errors for public expressions retain their detailed explanations.

Compilation and validation errors omit attribute values for both environment
and literal inputs. They report the affected block/field and a safe cause, such
as invalid URL escape, expected duration syntax, invalid regex capture name,
unknown template function, unknown reference, or allowed enum choices. URL,
duration, regex, and template compilation errors include the original expression
location. The CLI uses the same diagnostics and exits unsuccessfully for invalid
configuration. Config filenames and literal block labels remain visible; keep
secrets in values rather than in those identifiers. This diagnostic policy does
not alter successfully loaded values or skip field validation.

For local runs, load `.env` before invoking the proxy if needed; the CLI does
not load it automatically:

```sh
set -a; . ./.env; set +a
```

## Validation Rules

Startup fails on invalid configuration. Common checks include:

- missing listener or auth blocks
- more than one listener or auth block
- unsupported listener, credential, or target types
- duplicate client, credential, target, or parser names
- invalid parser config such as empty `prefix`, `bucket`, `pattern`, or `suffix`
- routes that reference unknown parsers or destinations
- invalid operation names or read preferences
- non-reversible rewrites on routes supporting `ListObjectsV2`
- duplicate route operations or destinations
- `on_match = "continue"` on a route that is not write-only
- `dispatch = "all"` without `PutObject` or `DeleteObject`
- `sigv4_static` auth without any clients
- virtual buckets that reference unknown routes

`CopyObject` is recognized for classification but explicitly rejected as unsupported in v1.
