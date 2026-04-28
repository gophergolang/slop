# Generated from VibeGuard Declaration Standard
# Uses vibeguard/go-secure-base (or local pattern)

FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tini && \
    addgroup -S -g 10001 appgroup && \
    adduser -S -u 10001 -G appgroup -h /app -s /sbin/nologin appuser
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app /app/app
USER appuser
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/app"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/health || exit 1

LABEL org.opencontainers.image.title="team-task-saas" \
      org.opencontainers.image.version="1.0.0" \
      security.non-root="true" \
      security.static-binary="true"