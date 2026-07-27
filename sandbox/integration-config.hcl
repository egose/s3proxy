listener "http" "public" {
  address = ":8082"

  addressing {
    path_style     = true
    virtual_hosted = false
  }
}

auth "main" {
  mode = "sigv4_static"

  client "ci" {
    access_key = env("S3PROXY_CLIENT_CI_ACCESS_KEY")
    secret_key = env("S3PROXY_CLIENT_CI_SECRET_KEY")
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
