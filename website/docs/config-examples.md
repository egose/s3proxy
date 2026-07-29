---
sidebar_position: 5
---

# Config Examples

This page collects complete configuration examples for common `s3proxy` setups.

Use them as starting points, then adapt endpoints, bucket names, and secrets for your environment.

## Local Development With One Backend

This is the smallest useful config. It exposes one path-based virtual bucket view and skips inbound auth.

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
    strip_path_prefix = "/images"
    bucket            = "images-store"
  }
}

bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
```

Use this when:

- you are testing locally
- the proxy is behind another trusted boundary
- one backend is enough

## Static SigV4 Auth Plus One Route

This adds inbound client authentication and limits the client to one route.

```hcl
listener "http" "public" {
  address = ":8080"

  addressing {
    path_style     = true
    virtual_hosted = false
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
  endpoint         = "http://127.0.0.1:9000"
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
    strip_path_prefix = "/images"
    bucket            = "images-store"
  }
}

bucket "images" {
  visible_name = "images"
  route        = "images_rw"
}
```

Use this when:

- you want the proxy to verify client SigV4 signatures
- client-facing credentials must differ from backend credentials
- you want a route-level allow-list

## Fan-Out Replication To Two Backends

This mirrors writes to a primary and replica backend.

```hcl
listener "http" "public" {
  address = ":8080"

  addressing {
    path_style     = true
    virtual_hosted = false
  }
}

auth "main" {
  mode = "sigv4_static"

  client "writer" {
    access_key      = env("S3PROXY_CLIENT_WRITER_ACCESS_KEY")
    secret_key      = env("S3PROXY_CLIENT_WRITER_SECRET_KEY")
    allow_routes    = ["route.replicate_rw"]
    visible_buckets = ["replicate"]
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
  endpoint         = "http://127.0.0.1:9000"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "primary"
}

target "s3" "replica" {
  endpoint         = "http://127.0.0.1:8333"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "replica"
}

parser "path_prefix" "replicate_prefix" {
  prefix = "/replicate"
}

route "replicate_rw" {
  parser          = "replicate_prefix"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject"]
  destinations    = ["primary", "replica"]
  dispatch        = "all"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix = "/replicate"
    bucket            = "testbucket"
  }
}

bucket "replicate" {
  visible_name = "replicate"
  route        = "replicate_rw"
}
```

Use this when:

- writes must land on every backend
- reads can still prefer one backend
- you understand the memory cost of buffering write bodies for replay

## Ordered Read Failover

This prefers one backend for reads and falls back only on transport failure, timeout, or upstream `5xx`.

```hcl
listener "http" "public" {
  address = ":8080"

  addressing {
    path_style     = true
    virtual_hosted = false
  }
}

auth "main" {
  mode = "sigv4_static"

  client "reader" {
    access_key      = env("S3PROXY_CLIENT_READER_ACCESS_KEY")
    secret_key      = env("S3PROXY_CLIENT_READER_SECRET_KEY")
    allow_routes    = ["route.failover_read"]
    visible_buckets = ["failover"]
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
  endpoint         = "http://10.0.0.10:9000"
  region           = "us-east-1"
  force_path_style = true
  timeout          = "250ms"
  credentials      = "primary"
}

target "s3" "replica" {
  endpoint         = "http://10.0.0.11:9000"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "replica"
}

parser "path_prefix" "failover_prefix" {
  prefix = "/failover"
}

route "failover_read" {
  parser          = "failover_prefix"
  operations      = ["GetObject", "HeadObject"]
  destinations    = ["primary", "replica"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "ordered_failover"

  rewrite {
    strip_path_prefix = "/failover"
    bucket            = "archive"
  }
}

bucket "failover" {
  visible_name = "failover"
  route        = "failover_read"
}
```

Use this when:

- you want a preferred backend and a warm backup
- `404` from the primary should stay a `404`
- target timeout should bound failover latency

## Virtual-Hosted Bucket Matching

This enables bucket addressing through the host header.

```hcl
listener "http" "public" {
  address = ":8080"

  addressing {
    path_style     = true
    virtual_hosted = true
    host_suffixes  = ["s3proxy.example.com"]
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
  endpoint         = "https://minio.internal"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "primary"
}

parser "bucket_exact" "assets" {
  bucket = "assets"
}

route "assets_rw" {
  parser          = "assets"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "ListObjectsV2", "HeadBucket"]
  destinations    = ["primary"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    bucket = "assets-prod"
  }
}

bucket "assets" {
  visible_name = "assets"
  route        = "assets_rw"
}
```

Use this when:

- clients address buckets through `bucket.s3proxy.example.com`
- you want the visible bucket name to differ from the backend bucket

## Tips

- keep client credentials and backend credentials separate
- use `visible_buckets` to control what `ListBuckets` returns
- prefer explicit target `timeout` values for failover routes
- avoid large fan-out writes unless you are comfortable buffering those bodies in memory
