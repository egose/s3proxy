# s3proxy

A path-based multi-backend proxy for S3-compatible APIs, written in Go.

The proxy accepts S3-compatible requests, optionally authenticates clients with
SigV4, determines one or more destination backends from the incoming request
path, bucket name, host, and operation type, rewrites requests as needed, and
forwards them to S3-compatible targets using the destinations' credentials.

## Current V1 Scope

### Supported Operations

- `GetObject`
- `HeadObject`
- `PutObject`
- `DeleteObject`
- `HeadBucket`
- `ListObjectsV2`
- `ListBuckets` (virtual/proxy-defined)

Supported v1 operations accept only the query keys needed for their current contract. Inbound SigV4 presign query keys (`X-Amz-Algorithm`, `X-Amz-Credential`, `X-Amz-Date`, `X-Amz-Expires`, `X-Amz-Security-Token`, `X-Amz-Signature`, and `X-Amz-SignedHeaders`) are accepted for authentication and are not forwarded to backends.

| Operation                                                                           | Supported non-auth query keys                                                                                                                            |
| ----------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GetObject`, `HeadObject`, `PutObject`, `DeleteObject`, `HeadBucket`, `ListBuckets` | none, except optional AWS SDK `x-id` matching the operation name                                                                                         |
| `ListObjectsV2`                                                                     | `list-type=2`, `continuation-token`, `delimiter`, `encoding-type`, `fetch-owner`, `max-keys`, `prefix`, `start-after`, and optional `x-id=ListObjectsV2` |

All other query-key combinations are rejected before route dispatch. This includes object or bucket subresources such as `acl`, `tagging`, `retention`, `legal-hold`, `torrent`, `versioning`, `versions`, `versionId`, `restore`, `select`, response header overrides such as `response-content-type`, and multipart query variants such as `uploads`, `uploadId`, and `partNumber`.

### Auth Modes

- `none` – skip inbound authentication (trusted environments only)
- `sigv4_static` – validate inbound SigV4 signatures with statically configured client credentials

Header authentication, presigned URLs, and outbound signing use S3 canonical URI rules: the escaped path is signed without a second round of URI escaping or path normalization. This corrects signature mismatches for spaces, literal percent characters, Unicode, and escaped separators in object keys. Custom clients using the AWS SDK v2 standalone signer must set `DisableURIPathEscaping = true`; signatures using the previous generic double-escaping behavior are not accepted for escaped paths.

### Routing

- path-style and virtual-hosted addressing
- parsers: path prefix, bucket exact, bucket regex, host suffix
- ordered route evaluation with `stop` / `continue` match modes
- destination dispatch modes: `first` (single target) and `all` (fan-out)
- read preferences: `first`, `random`, `hash`, `ordered_failover`

### Rewrite

- strip path prefix
- strip / prepend key prefix
- override bucket name
- key templates with captures from regex parsers

### Unsupported in V1

- Multipart uploads — return S3-compatible `NotImplemented`
- AWS streaming upload formats (`aws-chunked`, `STREAMING-` payload hashes, or `X-Amz-Trailer` declarations) — return `501 NotImplemented` before authentication or body reads. Use ordinary single-request uploads; HTTP transfer chunking and gzip object bytes remain supported. See the [payload contract](website/docs/api-reference.md#request-payload-formats).
- Unsupported S3 subresource operations — return S3-compatible `NotImplemented`
- Credential generation / rotation / DB-backed auth
- Merged multi-backend `ListObjects` pagination
- Hot config reload
- Full presigned URL feature parity

See [docs/design.md](docs/design.md) for the full design document.

## Install

### via asdf

Add the plugin:

```sh
asdf plugin add s3proxy
# or
asdf plugin add s3proxy https://github.com/egose/s3proxy.git
```

Install and activate a version:

```sh
# List all available versions
asdf list all s3proxy

# Install a specific version
asdf install s3proxy <version>

# Install the latest stable version
asdf install s3proxy latest

# Set the global version
asdf global s3proxy <version>
```

Once installed, the `s3proxy` binary is available directly on your `PATH`:

```sh
s3proxy serve --config /etc/s3proxy/config.hcl
s3proxy validate --config /etc/s3proxy/config.hcl
s3proxy routes --config /etc/s3proxy/config.hcl
s3proxy print-example-config
s3proxy version
```

Please check the [asdf documentation](https://github.com/asdf-vm/asdf) for more details.

Version discovery and downloads use the same strict SemVer grammar as release
validation, including versions such as `3.0.0-rc.1+build.4`. Discovery removes one
optional `v` prefix and deduplicates only identical complete versions. The list is
in ascending SemVer precedence: numeric major/minor/patch, numeric prerelease
identifiers before nonnumeric ones, numeric or ASCII lexical identifier order,
shorter matching identifier lists first, and stable versions after prereleases.
Build metadata has no SemVer precedence; equal-precedence versions are ordered by
their complete normalized strings in C-locale byte order (so `1.2.3` precedes
`1.2.3+A`, and `1.2.3+build.10` precedes `1.2.3+build.2`). Every variant is retained.
Downloads always use the corresponding `v`-prefixed release URL.

The shared `scripts/semver.sh` helper ships in the plugin checkout and is resolved
relative to each script, independent of the working directory. It uses Bash,
POSIX awk and ordinary lexical sort/cut; no Go, Node, Python or GNU `sort -V` is
needed. Numeric sort keys encode digit count as a unary prefix followed by the
original digits, avoiding integer overflow and floating-point rounding even for
long identifiers. Identifier separators sort before all valid identifier bytes,
and the complete version is the final tie-break. Git discovery must succeed
before any list is emitted. Linux tools are covered by the mocked workflow tests;
macOS/BSD implementations have not been executed locally.

The installer validates and extracts one regular binary into a temporary directory
inside the installation directory, sets mode `0755`, then publishes `bin/`. An
existing `bin` directory, file, or symlink (including a dangling symlink) is rejected. Downloaded archives
and checksums remain caller-owned and are not removed or modified by installation.
Failures before publication clean up staging and may leave an empty installation
directory. If removing the empty stage fails after publication, installation
reports failure even though the complete binary is already present; it does not
roll back caller-visible content. Cleanup is best-effort if filesystem removal
itself fails. Serialize installations and changes to their destination: the
portable existence check and `mv` are not a concurrent no-replace transaction.
Staging inside the destination keeps publication on the same filesystem even when
the installation directory is a mount point or a symlink to another filesystem.
Abrupt process death and crash durability are not transactional guarantees.

## Authenticated Starter From The Binary

No source checkout or environment setup is needed to print a version-matched starter:

```sh
s3proxy print-example-config > config.hcl
```

The shell creates the file; the command only writes static HCL to stdout. It accepts
no arguments or configuration flags, keeps literal `env("...")` references, and
does not load configuration, evaluate environment values, or start the runtime.
Output failures return a nonzero exit status.

The starter listens on `127.0.0.1:8080`, uses path-style addressing and
`sigv4_static`, and exposes only `images` through route `images_rw` to target
`primary` / pre-existing backend bucket `images-store` in `us-east-1`. Its explicit
permissions cover Get/Head/Put/DeleteObject, HeadBucket, ListObjectsV2, and local
virtual ListBuckets discovery. Client keys (`S3PROXY_CLIENT_ACCESS_KEY` and
`S3PROXY_CLIENT_SECRET_KEY`) are separate from backend keys
(`S3PROXY_TARGET_PRIMARY_ACCESS_KEY` and `S3PROXY_TARGET_PRIMARY_SECRET_KEY`);
`S3PROXY_TARGET_PRIMARY_ENDPOINT` supplies the backend URL.

After exporting those five variables, inspect and run it:

```sh
s3proxy validate --config ./config.hcl
s3proxy routes --config ./config.hcl
s3proxy serve --config ./config.hcl
```

The starter sets 32 MiB per-request / 256 MiB aggregate replay bounds, a 2-minute
request-read deadline, 2-minute complete upstream timeout, and 5-minute response-write
deadline. Tune these for object sizes, concurrency and link speeds; replay limits
are not universal upload-size limits. See the [authenticated native quickstart](website/docs/quickstart.md#authenticated-binary-only-starter)
for variable setup, prerequisites and timeout details. The separate manual
auth-none quickstart remains available there for trusted testing.

For containers, use an entrypoint override to print and change the listener to
`:8080` before serving, with restricted host publishing and TLS at the outer layer:
see [Docker starter onboarding](website/docs/deployment.md#authenticated-starter-in-docker).

## Example Configuration

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
    access_key = env("S3PROXY_CLIENT_CI_ACCESS_KEY")
    secret_key = env("S3PROXY_CLIENT_CI_SECRET_KEY")

    allow_routes = [
      "route.images_rw",
    ]

    visible_buckets = ["images"]
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

parser "bucket_regex" "tenant_logs" {
  pattern = "^tenant-(?P<tenant>[a-z0-9-]+)-logs$"
}

route "images_rw" {
  parser        = "images"
  operations    = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "ListObjectsV2"]
  destinations  = ["primary"]
  dispatch      = "first"
  on_match      = "stop"
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

`replay_body_max_bytes` limits how much request body the proxy will buffer per request when it needs a replayable body, such as `dispatch = "all"`, `on_match = "continue"`, authenticated payload-hash verification, or unknown-length outbound uploads. `0` uses the built-in default of `33554432` bytes (`32 MiB`). Oversized replay attempts fail with `413 EntityTooLarge`. `replay_body_aggregate_max_bytes` limits retained replay buffers across the process; `0` uses the built-in default of `268435456` bytes (`256 MiB`). Aggregate exhaustion fails immediately with `503 SlowDown` instead of blocking request goroutines.

## CLI

```sh
s3proxy serve --config /etc/s3proxy/config.hcl
s3proxy validate --config /etc/s3proxy/config.hcl
s3proxy routes --config /etc/s3proxy/config.hcl
s3proxy print-example-config
s3proxy version
```

`routes` (also `routes -c PATH`) fully validates the config and prints deterministic
JSON topology without constructing the app, binding a listener, contacting backends,
or emitting runtime logs. Routes have one-based ordinals in declaration order;
parser and destination references resolve to public declaration labels. Operation
and destination order is preserved, and `read_preference` defaults to `first`.
Operation lists are explicit: empty, omitted, or wildcard route filters are invalid.

The report contains only route ordinal/label, parser label/kind, operations,
destination labels, `dispatch`, `on_match`, and `read_preference`. It excludes
credentials, endpoints, regions, match strings, rewrites, visible bucket values,
environment names, and arbitrary environment values. Literal declaration labels
are public metadata: keep secrets out of labels and diagnostic file paths.
This reports configured topology, not simulated request routing, backend health,
or authorization. See the [schema and output example](website/docs/operations.md#offline-route-topology)
before using it in rollout tooling.

## Docker

```sh
docker build -t s3proxy .
docker run --rm \
  -p 8080:8080 \
  -v ./config.hcl:/etc/s3proxy/config.hcl:ro \
  -e S3PROXY_CLIENT_CI_ACCESS_KEY=... \
  -e S3PROXY_CLIENT_CI_SECRET_KEY=... \
  -e S3PROXY_TARGET_PRIMARY_ACCESS_KEY=... \
  -e S3PROXY_TARGET_PRIMARY_SECRET_KEY=... \
  s3proxy
```

## Local Run

Create `config.hcl` then:

```sh
go run ./cmd/s3proxy serve --config config.hcl
```

## Tests

```sh
go test ./...                                  # unit tests
make vet test                                  # vet + unit tests
make test-race                                # unit tests with the race detector
```

### Integration tests

The integration test suite (`internal/integration`, build-tagged `integration`)
exercises the proxy end-to-end against the sandbox docker-compose stack:
MinIO as the primary backend and SeaweedFS as the replica.

```sh
cp .env.example .env                            # then edit secrets if you want
make sandbox-integration-up                    # start stack + proxy + run tests + tear down
# or, to leave the stack up for iterative runs:
make sandbox-up DAEMON=true
make build
make test-integration                          # repeats against the running stack
make sandbox-down
make test-integration-race                      # fan-out path under the race detector
```

`make sandbox-integration-up` sources `.env` into the proxy's environment
(so `env("...")` config calls resolve), runs the stack + proxy +
tests + teardown in one shot, and exits with the test's status.

All sandbox helpers use the explicit Compose project `s3proxy-sandbox` via
`scripts/sandbox-compose.sh`, preferring `docker compose` and falling back to
`docker-compose`. Set `SANDBOX_PROJECT_NAME=s3proxy-my-worktree` consistently for
an alternate project (shell environment or Make command-line assignment).
`COMPOSE_PROJECT_NAME` does not override this identity. Container names and new
volumes are project-scoped; host ports still require one local stack at a time.
Run `make test-sandbox` for the Docker-free lifecycle regression.

Older versions used the shared project `sandbox`, which could remove another
repository's containers as orphans. Automatic orphan removal and global image
pruning have been removed. Existing containers/volumes are not migrated or
deleted automatically. See [sandbox isolation and migration](website/docs/operations.md#sandbox-isolation-and-migration)
before handling a legacy stack or a port conflict.

See `sandbox/integration-config.hcl` for the routing/parsers/credentials
configuration the suite expects.

## Environment Variables

Use `env("VAR")` in any string attribute in the HCL config to read an
environment variable. It is a native HCL function returning a literal string:
`${...}`, `%{...}`, quotes, newlines, backslashes, and Unicode in the value
are data, never HCL source or templates. Do not commit secret values into the
config file.

Ordinary function-call whitespace, such as `env ( "VAR" )`, is supported.
Text such as `env("VAR")` inside comments or literal strings is not evaluated.
An unset variable returns an empty string; required-field validation may then
reject it, while optional string fields may remain empty. Parse/decode errors
refer to the original config filename and source locations, including leading
blank lines. Export variables before invoking the CLI; `.env` is not loaded
automatically.

All `env()` results are treated as sensitive in configuration diagnostics,
including values used for endpoints, timeouts, patterns, templates, references,
and enums. Errors identify the block and field with a safe cause; they do not
echo attribute values during compilation or validation. HCL errors involving
`env()` withhold value-bearing details while retaining their diagnostic category
and original source location. Duplicate-object-key errors also withhold literal
key values. URL, duration, regex, and template compilation
errors also identify the original expression location. File paths and literal
block labels remain visible.

## Notes on Behavior

- `key_template` uses Go template syntax. Accessed data is keyed by `Bucket`,
  `Key`, and `Captures` (a `map[string]string` of regex named-group captures).
- Escaped path bytes are preserved through routing and rewrites, so keys such
  as `%2F` stay distinct from literal path separators.
- Prepending a prefix preserves every incoming key byte, including leading and
  repeated slashes: `assets/` maps `foo`, `/foo`, and `//foo` to `assets/foo`,
  `assets//foo`, and `assets///foo`. Only the prefix's optional join slash is
  normalized.
- Rewrites must preserve the authorized operation's shape. An object key that
  becomes empty returns `400 InvalidRequest` before any matched route is
  dispatched. `HeadBucket` applies only the bucket rewrite and addresses the
  backend bucket root, even when object keys use a template.
- Upstream redirects (`301`, `302`, `303`, `307`, `308`) are rejected without
  following or forwarding their `Location`. Configure the correct backend
  endpoint and region; the proxy does not discover regions through redirects.
  Rejection is a backend request failure, normally HTTP `502` (`InternalError`).
  Configured ordered failover may try the next configured destination.
- `ListObjectsV2` supports bucket-only rewrites, `prepend_key_prefix`, and
  prefix-only templates: literals/`.Bucket`/`.Captures.name` followed by exactly
  one terminal `{{ .Key }}`. Prepend and template prefixes can be combined.
  Listing routes reject all `strip_path_prefix`/`strip_key_prefix` rules and
  other templates at startup. Remove redundant bucket-name stripping from old
  examples: path-style parsing already separates the bucket from the key.
- List `prefix`/`start-after` and XML bucket/key fields are translated into the
  virtual namespace, including `encoding-type=url`. Configured prefixes use raw
  path notation with valid percent escapes (`%25` for a literal percent);
  query/XML keys are decoded S3 keys, not raw paths. Missing captures or invalid
  runtime prefix escapes fail with `400 InvalidRequest` before dispatch.
- Listings use one backend; continuation tokens are opaque and unmodified.
  `random` or `ordered_failover` can switch backends between pages, making tokens
  invalid or results inconsistent. Use a fixed backend for stable pagination;
  no merged listings or backend-pinned cursors are provided.
- Successful list XML is buffered and transformed with 8 MiB input/output,
  50,000-element, four-level XML depth, and five-second transformation limits
  (or earlier request/target cancellation). Malformed, oversized, unsupported
  XML or out-of-namespace results fail closed with HTTP `502` (`InternalError`); no partial
  listing is sent. Objects exactly equal to the hidden namespace prefix cannot
  represent a nonempty visible key and also fail closed. GET object bodies stay
  streaming. See [listing semantics](docs/design.md#listing-semantics).
- For multi-destination routes, `read_preference` controls which destination
  is used for reads (`first` by default).
- For multi-destination writes with `dispatch = "all"`, all destinations must
  succeed. Upstream HTTP failures return the primary upstream error response;
  transport or replay failures return a proxy error. Fan-out is not
  transactional: a destination that succeeds before another destination fails is
  not rolled back.
- For multi-route writes with `on_match = "continue"`, every matched route must
  succeed; a later route failure cannot be hidden by an earlier success.
- `ordered_failover` only fails over on transport errors, timeouts, and
  upstream `5xx`. It does not fail over on `404` / `NoSuchKey` /
  `NoSuchBucket` responses.
- `target "s3"` supports an optional `timeout` duration. For
  `ordered_failover`, this directly bounds how long the proxy waits before
  failing over from a slow or unreachable backend.
- `ListBuckets` returns proxy-defined virtual buckets, not upstream discovery.
- Request body replay for multi-destination writes, multi-route writes, payload-hash
  verification, and unknown-length outbound uploads is bounded per request by
  `listener.replay_body_max_bytes` and across the process by
  `listener.replay_body_aggregate_max_bytes`. Per-request overflow fails with
  `413 EntityTooLarge`; aggregate exhaustion fails with `503 SlowDown`.

## Deferred / Planned

See the "Deferred Features" section in [docs/design.md](docs/design.md) for a
full list of intentionally deferred features, including multipart upload
support and dynamic credential generation.
