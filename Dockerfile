# syntax=docker/dockerfile:1

FROM golang:1.26.5@sha256:705e964a93a2fd2e75c7d59bb7d781b57e30f12293ffde5175c69229e18fb678 AS builder

ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -buildvcs=false -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/s3proxy \
    ./cmd/s3proxy

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="s3proxy" \
      org.opencontainers.image.description="Path-based proxy for S3-compatible APIs" \
      org.opencontainers.image.source="https://github.com/egose/s3proxy" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$REVISION"

COPY --from=builder /out/s3proxy /usr/local/bin/s3proxy

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/s3proxy", "serve", "--config", "/etc/s3proxy/config.hcl"]
