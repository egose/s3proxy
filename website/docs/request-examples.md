---
sidebar_position: 6
---

# Request Examples

These examples use AWS CLI against `s3proxy`.

For `sigv4_static`, the proxy validates the client's SigV4 signature. AWS CLI is the simplest way to generate those signed requests without hand-rolling the `Authorization` header.

## Common Setup

Set your proxy endpoint and client credentials:

```sh
export S3_ENDPOINT=http://127.0.0.1:8080
export AWS_ACCESS_KEY_ID=$S3PROXY_CLIENT_ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=$S3PROXY_CLIENT_SECRET_KEY
export AWS_DEFAULT_REGION=us-east-1
```

If your proxy is running with `mode = "none"`, AWS CLI still expects credentials locally, so any placeholder values are fine.

## List Virtual Buckets

`ListBuckets` returns the buckets exposed by proxy config and filtered by `visible_buckets`.

```sh
aws --endpoint-url "$S3_ENDPOINT" s3api list-buckets
```

## Upload An Object

This request targets the proxy-visible bucket `images`:

```sh
aws --endpoint-url "$S3_ENDPOINT" s3api put-object \
  --bucket images \
  --key cat.jpg \
  --body ./cat.jpg
```

If the matching route rewrites `bucket = "images-store"`, the object is stored in the backend bucket `images-store` even though the client addressed `images`.

## Read The Object Back

```sh
aws --endpoint-url "$S3_ENDPOINT" s3api get-object \
  --bucket images \
  --key cat.jpg \
  ./cat.out.jpg
```

For a route with `read_preference = "ordered_failover"`, the proxy tries later destinations on errors returned while preparing, signing, or sending the upstream request, including transport errors, timeouts, replay-limit errors, and upstream `5xx`. It does not fail over on an upstream `4xx` response.

## Head An Object

```sh
aws --endpoint-url "$S3_ENDPOINT" s3api head-object \
  --bucket images \
  --key cat.jpg
```

## Delete An Object

```sh
aws --endpoint-url "$S3_ENDPOINT" s3api delete-object \
  --bucket images \
  --key cat.jpg
```

On a route with `dispatch = "all"`, `DeleteObject` is replayed to every configured destination.

## List Objects In A Bucket View

```sh
aws --endpoint-url "$S3_ENDPOINT" s3api list-objects-v2 \
  --bucket images
```

In v1, `ListObjectsV2` always comes from one effective backend. The proxy does not merge pagination across multiple destinations.

## Head A Bucket

```sh
aws --endpoint-url "$S3_ENDPOINT" s3api head-bucket \
  --bucket images
```

## Virtual-Hosted Request Example

If the listener enables `virtual_hosted = true` with `host_suffixes = ["s3proxy.example.com"]`, a client can address a bucket through the host instead of the path.

Example request shape:

```http
GET /cat.jpg HTTP/1.1
Host: images.s3proxy.example.com
Authorization: AWS4-HMAC-SHA256 ...
```

## Unsupported Operations

Multipart upload operations are not implemented in v1. `CopyObject` is also rejected.

S3 subresource query operations are rejected before route dispatch unless they are part of the documented supported query surface. Examples include `?acl`, `?tagging`, `?retention`, `?legal-hold`, `?versionId=...`, `?restore`, `?select`, response header overrides, and multipart query variants such as `?uploads` and `?uploadId=...`.

The proxy returns an S3-compatible `NotImplemented` error for those requests instead of attempting partial support.

## Behavior Notes

- direct single-destination reads do not fail over anywhere else
- `dispatch = "all"` writes must succeed on every destination
- `on_match = "continue"` writes must succeed on every matched route
- `ordered_failover` does not fail over on `404`, `NoSuchKey`, or `NoSuchBucket`
- request paths preserve escaped bytes, so `%2F` remains distinct from `/`
