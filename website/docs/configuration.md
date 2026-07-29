---
sidebar_position: 3
---

# Configuration

`s3proxy` uses two-label HCL blocks. The core building blocks are:

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
- enabling `virtual_hosted` requires at least one `host_suffix`

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

Example:

```hcl
auth "main" {
  mode = "sigv4_static"

  client "admin" {
    access_key      = env("S3PROXY_CLIENT_ADMIN_ACCESS_KEY")
    secret_key      = env("S3PROXY_CLIENT_ADMIN_SECRET_KEY")
    allow_routes    = ["*"]
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

Supported `read_preference` values:

- `first`
- `random`
- `hash`
- `ordered_failover`

For `ordered_failover`, failover happens only on transport errors, timeouts, and upstream `5xx` responses.

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
  strip_path_prefix  = "/images"
  prepend_key_prefix = "assets/"
  bucket             = "images-store"
  key_template       = "{{ .Captures.tenant }}/{{ .Key }}"
}
```

Template data uses the names `Bucket`, `Key`, and `Captures`.

## Virtual Buckets

`ListBuckets` returns buckets you define explicitly:

```hcl
bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
```

`visible_name` is what the client sees. `route` decides how requests for that bucket are handled.

## Environment Variables

Use `env("VAR")` anywhere a string is allowed. The value is textually inlined before HCL parsing.

For local runs, load `.env` before invoking the proxy if needed:

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
- `sigv4_static` auth without any clients
- virtual buckets that reference unknown routes

`CopyObject` is recognized for classification but explicitly rejected as unsupported in v1.
