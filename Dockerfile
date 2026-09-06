# syntax=docker/dockerfile:1

# Build stage. Always runs on the native build platform and cross-compiles to the
# target, so multi-arch builds never go through QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the source code
COPY . .

# Build the application (fully static, no cgo) for the requested target platform
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build,id=go-build-$TARGETOS-$TARGETARCH \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/proxy ./cmd/proxy/main.go

# Prepare the writable database directory owned by the distroless nonroot user (65532),
# since the final stage has no shell to run mkdir/chown.
RUN mkdir -p /out/database && chown -R 65532:65532 /out

# Run stage: distroless static, nonroot (uid/gid 65532).
# Ships ca-certificates and tzdata; contains no shell or package manager.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy the binary and the database directory from the builder stage
COPY --from=builder --chown=65532:65532 /out/proxy /app/proxy
COPY --from=builder --chown=65532:65532 /out/database /app/database

USER nonroot:nonroot

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
ENTRYPOINT ["/app/proxy"]
CMD ["--use-cache=true", "--db-path=/app/database/metrics.db"]
