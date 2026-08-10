listener "http" "public" {
  address = ":8082"
  replay_body_max_bytes = 64
  replay_body_aggregate_max_bytes = 268435456

  addressing {
    path_style     = true
    virtual_hosted = true
    host_suffixes  = ["s3proxy.test"]
  }
}

auth "main" {
  mode = "sigv4_static"

  client "ci" {
    access_key      = env("S3PROXY_CLIENT_CI_ACCESS_KEY")
    secret_key      = env("S3PROXY_CLIENT_CI_SECRET_KEY")
    allow_routes    = ["*"]
    visible_buckets = ["*"]
  }
}

// Backends: MinIO is the primary, SeaweedFS is the replica.
// Each target's credentials are independent.

credential "static" "primary" {
  access_key = env("S3PROXY_TARGET_PRIMARY_ACCESS_KEY")
  secret_key = env("S3PROXY_TARGET_PRIMARY_SECRET_KEY")
}

credential "static" "replica" {
  access_key = env("S3PROXY_TARGET_REPLICA_ACCESS_KEY")
  secret_key = env("S3PROXY_TARGET_REPLICA_SECRET_KEY")
}

credential "static" "forbidden" {
  access_key = "forbidden-ak"
  secret_key = "forbidden-sk" // pragma: allowlist secret
}

target "s3" "primary" {
  endpoint         = "http://localhost:9000"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "primary"
}

target "s3" "replica" {
  endpoint         = "http://localhost:8333"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "replica"
}

target "s3" "missing" {
  endpoint         = "http://localhost:65534"
  region           = "us-east-1"
  force_path_style = true
  timeout          = "250ms"
  credentials      = "primary"
}

target "s3" "forbidden" {
  endpoint         = "http://localhost:9000"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "forbidden"
}

target "s3" "error500" {
  endpoint         = "http://localhost:18081"
  region           = "us-east-1"
  force_path_style = true
  credentials      = "primary"
}

// Route /primary/* to MinIO only. Validates single-destination forwarding
// against a real S3 backend.

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

// Route /replica/* to SeaweedFS only. Proves we can target a different
// S3-compatible backend (different credentials, different endpoint).

parser "path_prefix" "replica_prefix" {
  prefix = "/replica"
}

route "replica_only" {
  parser          = "replica_prefix"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "HeadBucket", "ListObjectsV2"]
  destinations    = ["replica"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix = "/replica"
    bucket            = "testbucket"
  }
}

// Route /replicate/* to BOTH backends. dispatch = "all" means writes fan out
// to every destination; reads use read_preference = "first" to pick one
// effective target. Validates multi-destination dispatch against real
// backends.

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

parser "path_prefix" "failover_prefix" {
  prefix = "/failover"
}

route "replica_failover_read" {
  parser          = "failover_prefix"
  operations      = ["GetObject", "HeadObject"]
  destinations    = ["missing", "replica"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "ordered_failover"

  rewrite {
    strip_path_prefix = "/failover"
    bucket            = "testbucket"
  }
}

parser "path_prefix" "failover_500_prefix" {
  prefix = "/failover5xx"
}

route "failover_after_5xx" {
  parser          = "failover_500_prefix"
  operations      = ["GetObject", "HeadObject"]
  destinations    = ["error500", "replica"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "ordered_failover"

  rewrite {
    strip_path_prefix = "/failover5xx"
    bucket            = "testbucket"
  }
}

parser "path_prefix" "failover_404_prefix" {
  prefix = "/failover404"
}

route "failover_stops_on_404" {
  parser          = "failover_404_prefix"
  operations      = ["GetObject", "HeadObject"]
  destinations    = ["primary", "replica"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "ordered_failover"

  rewrite {
    strip_path_prefix = "/failover404"
    bucket            = "testbucket"
  }
}

parser "path_prefix" "compose_prefix" {
  prefix = "/compose"
}

route "compose_primary" {
  parser       = "compose_prefix"
  operations   = ["PutObject", "DeleteObject"]
  destinations = ["primary"]
  dispatch     = "first"
  on_match     = "continue"

  rewrite {
    strip_path_prefix = "/compose"
    bucket            = "testbucket"
  }
}

route "compose_replica" {
  parser       = "compose_prefix"
  operations   = ["PutObject", "DeleteObject"]
  destinations = ["replica"]
  dispatch     = "first"
  on_match     = "stop"

  rewrite {
    strip_path_prefix = "/compose"
    bucket            = "testbucket"
  }
}

parser "bucket_exact" "virtual_bucket" {
  bucket = "virtual-bucket"
}

route "virtual_hosted_primary" {
  parser          = "virtual_bucket"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject"]
  destinations    = ["primary"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    bucket = "testbucket"
  }
}

parser "bucket_regex" "tenant_logs" {
  pattern = "^tenant-(?P<tenant>[a-z0-9-]+)-logs$"
}

route "tenant_log_rewrite" {
  parser          = "tenant_logs"
  operations      = ["GetObject", "HeadObject", "PutObject", "DeleteObject"]
  destinations    = ["primary"]
  dispatch        = "first"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    bucket       = "testbucket"
    key_template = "{{ .Captures.tenant }}/{{ .Key }}"
  }
}

parser "path_prefix" "fanout_primary_http_fail_prefix" {
  prefix = "/fanout-primary-http-fail"
}

route "fanout_primary_http_fail" {
  parser          = "fanout_primary_http_fail_prefix"
  operations      = ["PutObject", "DeleteObject"]
  destinations    = ["forbidden", "primary"]
  dispatch        = "all"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix = "/fanout-primary-http-fail"
    bucket            = "testbucket"
  }
}

parser "path_prefix" "fanout_replica_http_fail_prefix" {
  prefix = "/fanout-replica-http-fail"
}

route "fanout_replica_http_fail" {
  parser          = "fanout_replica_http_fail_prefix"
  operations      = ["PutObject", "DeleteObject"]
  destinations    = ["primary", "forbidden"]
  dispatch        = "all"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix = "/fanout-replica-http-fail"
    bucket            = "testbucket"
  }
}

parser "path_prefix" "fanout_transport_fail_prefix" {
  prefix = "/fanout-transport-fail"
}

route "fanout_transport_fail" {
  parser          = "fanout_transport_fail_prefix"
  operations      = ["PutObject", "DeleteObject"]
  destinations    = ["primary", "missing"]
  dispatch        = "all"
  on_match        = "stop"
  read_preference = "first"

  rewrite {
    strip_path_prefix = "/fanout-transport-fail"
    bucket            = "testbucket"
  }
}

// Virtual buckets exposed to authenticated clients via ListBuckets.

bucket "primary_visible" {
  visible_name = "primary-bucket"
  route        = "primary_only"
}

bucket "replica_visible" {
  visible_name = "replica-bucket"
  route        = "replica_only"
}

bucket "replicate_visible" {
  visible_name = "replicate-bucket"
  route        = "replicate_rw"
}

bucket "failover_visible" {
  visible_name = "failover-bucket"
  route        = "replica_failover_read"
}

bucket "compose_visible" {
  visible_name = "compose-bucket"
  route        = "compose_primary"
}

bucket "virtual_hosted_visible" {
  visible_name = "virtual-bucket"
  route        = "virtual_hosted_primary"
}
