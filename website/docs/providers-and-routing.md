---
sidebar_position: 4
---

# Routing and Rewrites

`s3proxy` routes requests from a normalized S3 request context, not just from the raw HTTP method.

That matters because the proxy can match on path, bucket, host, and operation, then rewrite the outgoing bucket and key before signing the request for the destination backend.

## Addressing Modes

The proxy supports both S3 addressing styles:

- path-style requests such as `/bucket/key`
- virtual-hosted requests such as `bucket.s3proxy.example.com/key`

Enable them in the listener:

```hcl
listener "http" "public" {
  address = ":8080"

  addressing {
    path_style     = true
    virtual_hosted = true
    host_suffixes  = ["s3proxy.example.com"]
  }
}
```

## Parser Types

Routes reference a parser. Supported parser kinds are:

- `path_prefix`
- `bucket_exact`
- `bucket_regex`
- `host_suffix`

Examples:

```hcl
parser "path_prefix" "images" {
  prefix = "/images"
}

parser "bucket_regex" "tenant_logs" {
  pattern = "^tenant-(?P<tenant>[a-z0-9-]+)-logs$"
}

parser "host_suffix" "public_hosts" {
  suffix = "s3proxy.example.com"
}
```

`bucket_regex` can capture named groups, which are then exposed to rewrite templates.

## Route Evaluation Order

Routes are evaluated in config order.

Each route has an `on_match` policy:

- `stop`: stop evaluating routes after the match
- `continue`: keep evaluating later routes and collect more matches

This lets you compose behavior. For example, one `PutObject` can be applied to multiple matching routes if the earlier match uses `continue`.

For write requests matched by multiple routes, every matched route must succeed. A later route failure is returned as failure rather than hiding behind an earlier success.

## Dispatch Modes

Each route also has a `dispatch` policy:

- `first`: use a single destination
- `all`: send the write to every destination

`dispatch = "all"` is for multi-destination writes such as replication.

In v1:

- writes can fan out
- reads never fan out
- if any destination fails during a fan-out write, the request fails
- upstream HTTP failures preserve the primary upstream error response when available
- transport or replay failures return a proxy-generated failure

## Read Preference

When a route has more than one destination, reads choose one effective backend using `read_preference`:

- `first`
- `random`
- `hash`
- `ordered_failover`

`ordered_failover` is the safest choice when you want a preferred backend with a backup.

Failover happens only on:

- transport errors
- request timeouts
- upstream `5xx`

Failover does not happen on:

- `404`
- `NoSuchKey`
- `NoSuchBucket`

That behavior prevents the proxy from hiding routing mistakes or backend inconsistency.

## Rewrite Rules

After route selection, the proxy can rewrite the request before it builds the outbound backend request.

Supported rewrite fields:

- `strip_path_prefix`
- `strip_key_prefix`
- `prepend_key_prefix`
- `bucket`
- `key_template`

Example:

```hcl
route "tenant_logs" {
  parser          = "tenant_logs"
  operations      = ["GetObject", "PutObject", "DeleteObject", "ListObjectsV2"]
  destinations    = ["primary"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    bucket       = "shared-logs"
    key_template = "{{ .Captures.tenant }}/{{ .Key }}"
  }
}
```

Template data includes:

- `Bucket`
- `Key`
- `Captures`

## Strict Prefix Matching

`path_prefix` matching is intentionally strict.

If the prefix is `/replica`, a match happens only when:

- `RawPath == "/replica"`
- or `RawPath` starts with `"/replica/"`

`/replicate/...` does not match `/replica`.

## Virtual Buckets And Routing

`ListBuckets` is proxy-defined, not upstream-defined.

That means you can expose a clean bucket catalog even when the proxy is routing requests by path prefix or rewriting them into completely different backend buckets.

Example:

```hcl
bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
```

The visible bucket name presented to the client does not need to equal the backend bucket used after rewrite.

## Common Patterns

Single backend, path-based view:

```hcl
parser "path_prefix" "primary_prefix" {
  prefix = "/primary"
}

route "primary_only" {
  parser          = "primary_prefix"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "HeadBucket", "ListObjectsV2"]
  destinations    = ["primary"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix = "/primary"
    bucket            = "testbucket"
  }
}
```

Replicated writes:

```hcl
route "replicate_rw" {
  parser          = "replicate_prefix"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject"]
  destinations    = ["primary", "replica"]
  dispatch        = "all"
  on_match        = "stop"
  read_preference = "first"
}
```

Preferred backend with failover:

```hcl
route "replica_failover_read" {
  parser          = "failover_prefix"
  operations      = ["GetObject", "HeadObject"]
  destinations    = ["missing", "replica"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "ordered_failover"
}
```
