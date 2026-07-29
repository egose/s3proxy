## Unreleased

### Features

* add configurable replay body limits for multi-route, fan-out, and unknown-length outbound request replay

### Bug Fixes

* enforce listener addressing modes, tighten dispatch=all validation, and return signature-specific auth failures

### Documentation

* document `replay_body_max_bytes` and `413 EntityTooLarge` replay behavior across README and docs

## 0.0.1 (2026-07-28)

### Features

* accept unsigned presigned headers and preserve request path semantics ([db61e67](https://github.com/egose/s3proxy/commit/db61e67e4d628b46991fe28e830dafdfb745d1e3))
* add asdf release support and automated release workflow ([48fe2c3](https://github.com/egose/s3proxy/commit/48fe2c35ea5de3aaaad544e411eb9166298a439f))
* add docker image publish workflow ([538d0ec](https://github.com/egose/s3proxy/commit/538d0ec178ad3a4d73726e0938a1b98b90276fe3))
* add proxy CLI, workflows, and sandbox tooling ([dd806ee](https://github.com/egose/s3proxy/commit/dd806ee5b2164c23e30ce6b59aa0b8e3c4e2ab73))
* add request timeouts, presigned auth, and replayable routing ([e4003ee](https://github.com/egose/s3proxy/commit/e4003ee7594d4c04313cbf33348918ee7c02a1f2))
* improve host suffix routing matching ([a988efa](https://github.com/egose/s3proxy/commit/a988efa3703ccb2f66233603fe665978047ddf21))
* install a fixed Docker Compose version in the setup action ([bae2efe](https://github.com/egose/s3proxy/commit/bae2efe475de9106d9388983f9e9f3cd936f0825))
* install Docker Compose v2 in setup tools action ([354d077](https://github.com/egose/s3proxy/commit/354d0770dbbff6dfc38b588f805b41720ea7752e))
* tighten SigV4 verification and endpoint validation ([394f9bf](https://github.com/egose/s3proxy/commit/394f9bf2eed21b6cc0d835ce2642cc529a7fffc1))
* wait for sandbox services before running integration setup ([276e610](https://github.com/egose/s3proxy/commit/276e610a073a05e12dadd626066a7d4812c7d486))

### Bug Fixes

* clear successful fan-out primaries on partial failure ([e73631c](https://github.com/egose/s3proxy/commit/e73631cb2b7a899cf95cb97709ae2424cbbe5977))

### Documentation

* document asdf installation ([afe45ef](https://github.com/egose/s3proxy/commit/afe45efad6692e60a2a267124d6237ecc0d20ce4))
