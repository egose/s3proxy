## [0.3.1](https://github.com/egose/s3proxy/compare/v0.3.0...v0.3.1) (2026-08-23)

## [0.3.0](https://github.com/egose/s3proxy/compare/v0.2.0...v0.3.0) (2026-08-23)

### Features

* **website:** enforce v1 query contracts and harden S3 proxy lifecycle ([6d5098f](https://github.com/egose/s3proxy/commit/6d5098f037892db816050e87b5051002d0876f21))

### Bug Fixes

* **website:** handle presigned auth before rejecting unsupported operations ([634bcfb](https://github.com/egose/s3proxy/commit/634bcfbc6b0956f873d52ea6bbaf6c4d7fcc1b1d))

### Documentation

* **website:** refresh website layout spacing and theme component formatting ([296da76](https://github.com/egose/s3proxy/commit/296da76e65de04268c1337194b0d1a1713278023))
* **website:** update website docs for v1 routing, auth, and runtime behavior ([7b530b4](https://github.com/egose/s3proxy/commit/7b530b4fd0020ec28170dc376bb9361c41dd3497))

## [0.2.0](https://github.com/egose/s3proxy/compare/v0.1.0...v0.2.0) (2026-08-10)

### Features

* **website:** add replay budget, auth verification, and routing fixes ([4bc6802](https://github.com/egose/s3proxy/commit/4bc6802adf122a2e34a853673653c2e69cfd81a9))
* **website:** complete lifecycle, routing, logging, and integration updates ([b0d86da](https://github.com/egose/s3proxy/commit/b0d86dada8f1794ac72ccc9c5cdc9efb8fbc3a71))

### Documentation

* add codebase health remediation plan ([3007031](https://github.com/egose/s3proxy/commit/3007031d6ed82083faefcec3fc79614c592c2a05))

## [0.1.0](https://github.com/egose/s3proxy/compare/v0.0.1...v0.1.0) (2026-07-29)

### Features

* **website:** add Docusaurus website and documentation ([1bba1d2](https://github.com/egose/s3proxy/commit/1bba1d2df28c78e6291e2e6b11e3843cf0cd921e))
* **website:** add replay body limits and shared replay helpers ([036201f](https://github.com/egose/s3proxy/commit/036201fb6c09cb98fbf83c5df18cd24bf59448ee))
* **website:** document replay limits and listener addressing behavior ([ecfbc29](https://github.com/egose/s3proxy/commit/ecfbc2953b976e4333291a661de00292fa5f698c))

### Bug Fixes

* allow dispatch all for mixed read and write routes ([f2b6289](https://github.com/egose/s3proxy/commit/f2b628926f87389baa79543b02fbf0e49050f039))
* **website:** return signature mismatches and invalid addressing errors ([32484fe](https://github.com/egose/s3proxy/commit/32484fe8be1ea3074b7b442accd01454c2239e41))

## [0.0.1](https://github.com/egose/s3proxy/compare/dd806ee5b2164c23e30ce6b59aa0b8e3c4e2ab73...v0.0.1) (2026-07-28)

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
