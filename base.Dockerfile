# VibeGuard Secure Go Base Image (Go Edition)
# Version: 0.2
# Purpose: Minimal attack surface, static binary, non-root, production-ready base for all VibeGuard-generated Go applications
# Usage in generated projects: FROM ghcr.io/your-org/vibeguard/go-secure-base:0.2 (once published)
# Or copy this pattern directly into your generated Dockerfile

# =============================================================================
# BUILDER STAGE (Go 1.22, CGO disabled for fully static binary)
# =============================================================================
FROM golang:1.22-alpine AS builder

# Install build dependencies (git for private modules if needed, ca-certificates for HTTPS)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Create non-root user for build (good practice)
RUN addgroup -S builder && adduser -S builder -G builder

# Copy go.mod and go.sum first for better layer caching
COPY --chown=builder:builder go.mod go.sum ./
RUN go mod download

# Copy full source and build static binary
COPY --chown=builder:builder . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /vibeguard-app \
    ./cmd/server

# =============================================================================
# RUNTIME STAGE (minimal, secure, non-root)
# =============================================================================
FROM alpine:3.19 AS runtime

# Security hardening: minimal packages, update, clean
RUN apk add --no-cache ca-certificates tini \
    && rm -rf /var/cache/apk/*

# Create dedicated non-root user (UID 10001 is common and safe)
RUN addgroup -S -g 10001 appgroup && \
    adduser -S -u 10001 -G appgroup -h /app -s /sbin/nologin appuser

WORKDIR /app

# Copy the static binary from builder
COPY --from=builder --chown=appuser:appgroup /vibeguard-app /app/app

# Use non-root user
USER appuser

# Use tini as init system for proper signal handling and zombie reaping
ENTRYPOINT ["/sbin/tini", "--"]

# Default command (override in actual generated Dockerfile or docker-compose)
CMD ["/app/app"]

# Healthcheck (override in actual app with your /health endpoint)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/health || exit 1

# Recommended runtime security (applied via docker run / Kubernetes):
# --read-only
# --security-opt no-new-privileges:true
# --cap-drop ALL
# --security-opt seccomp=unconfined (or custom profile)
# Network: only expose necessary ports, use NetworkPolicy in K8s

# Labels for traceability and security scanning
LABEL org.opencontainers.image.title="VibeGuard Go Secure Base" \
      org.opencontainers.image.description="Hardened, static, non-root Go 1.22 base image for secure AI-generated applications" \
      org.opencontainers.image.version="0.2" \
      org.opencontainers.image.vendor="VibeGuard" \
      security.non-root="true" \
      security.static-binary="true" \
      security.minimal="true"

# =============================================================================
# ALTERNATIVE ULTRA-SECURE RUNTIME (uncomment to use distroless)
# =============================================================================
# FROM gcr.io/distroless/static:nonroot
# COPY --from=builder --chown=nonroot:nonroot /vibeguard-app /app/app
# USER nonroot
# ENTRYPOINT ["/app/app"]
# (Note: distroless has no shell, no package manager — maximum security, minimum surface)

# =============================================================================
# NOTES FOR GENERATED PROJECTS
# =============================================================================
# 1. In the actual generated Dockerfile, extend this pattern:
#    FROM golang:1.22-alpine AS builder
#    ... (same hardening + build)
#    FROM alpine:3.19
#    ... (same user + tini)
#    COPY --from=builder ... /app/app
#    USER appuser
#    ENTRYPOINT ["/sbin/tini", "--"]
#    CMD ["/app/app"]
#
# 2. For even smaller & more secure images, switch the final stage to:
#    FROM gcr.io/distroless/static:nonroot
#
# 3. Always run `govulncheck`, `gosec`, and `trivy image` on the final image.
# 4. Binary is fully static — no CGO, no glibc dependencies.
# =============================================================================