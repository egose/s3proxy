---
sidebar_position: 2
---

# Quickstart

This quickstart runs `s3proxy` locally against one S3-compatible backend with inbound auth disabled.

## Before You Start

You need:

- an S3-compatible backend such as MinIO
- backend credentials with access to a bucket you want to expose
- either an installed `s3proxy` binary or a local checkout of this repo

## Install

Install with `asdf`:

```sh
asdf plugin add s3proxy
# or
asdf plugin add s3proxy https://github.com/egose/s3proxy.git

asdf install s3proxy latest
asdf global s3proxy latest
```

Or build from source:

```sh
make build
./dist/s3proxy version
```

## Minimal Config

Create `config.hcl`:

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

With that config, requests for `/images/...` are rewritten into the backend bucket `images-store`.

## Export Secrets And Validate

If your config uses `env("...")`, export those variables before running the CLI:

```sh
export S3PROXY_TARGET_PRIMARY_ENDPOINT=http://127.0.0.1:9000
export S3PROXY_TARGET_PRIMARY_ACCESS_KEY=minioadmin
export S3PROXY_TARGET_PRIMARY_SECRET_KEY=minioadmin
```

If you keep them in `.env`, load them first:

```sh
set -a; . ./.env; set +a
```

Validate the config:

```sh
s3proxy validate --config ./config.hcl
```

Start the proxy:

```sh
s3proxy serve --config ./config.hcl
```

From source, the equivalent command is:

```sh
go run ./cmd/s3proxy serve --config ./config.hcl
```

## Send Requests

For trusted local testing with `mode = "none"`, the proxy skips inbound authentication. AWS CLI still needs credentials locally, so provide any placeholder values:

```sh
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
```

Upload an object through the proxy:

```sh
aws --endpoint-url http://127.0.0.1:8080 s3api put-object \
  --bucket images \
  --key hello.txt \
  --body ./hello.txt
```

List objects through the same route:

```sh
aws --endpoint-url http://127.0.0.1:8080 s3api list-objects-v2 \
  --bucket images
```

List the virtual buckets exposed by the proxy:

```sh
aws --endpoint-url http://127.0.0.1:8080 s3api list-buckets
```

## Switch To SigV4 Auth

For anything outside a trusted environment, use `sigv4_static` so the proxy verifies the caller's S3 SigV4 signature.

```hcl
auth "main" {
  mode = "sigv4_static"

  client "local-dev" {
    access_key      = env("S3PROXY_CLIENT_ACCESS_KEY")
    secret_key      = env("S3PROXY_CLIENT_SECRET_KEY")
    allow_routes    = ["route.images_rw"]
    visible_buckets = ["images"]
  }
}
```

Then point your S3 client or AWS CLI at the proxy with those client credentials:

```sh
export AWS_ACCESS_KEY_ID=$S3PROXY_CLIENT_ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=$S3PROXY_CLIENT_SECRET_KEY
```

The client credentials used to call the proxy are separate from the backend credentials used by `target "s3"`.

## Next Steps

- Add more routes and rewrites in [Configuration](./configuration.md)
- Set up fan-out replication or failover in [Config Examples](./config-examples.md)
- Review exact behavior in [API Reference](./api-reference.md)
