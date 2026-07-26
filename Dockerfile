# syntax=docker/dockerfile:1

FROM golang:1.26 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o /out/s3proxy \
    ./cmd/s3proxy

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/s3proxy /usr/local/bin/s3proxy

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/s3proxy", "serve", "--config", "/etc/s3proxy/config.hcl"]
