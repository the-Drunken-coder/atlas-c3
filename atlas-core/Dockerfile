FROM golang:1.25 AS build
ARG TARGETARCH=amd64
ARG TARGETOS=linux
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/atlas-core ./cmd/atlas-core

FROM alpine:3.21
LABEL atlas.core.project=atlas-core
WORKDIR /app
RUN apk add --no-cache ca-certificates su-exec wget \
 && addgroup -S -g 10001 atlas \
 && adduser -S -D -H -u 10001 -G atlas atlas \
 && mkdir -p /app /data/object-storage \
 && chown -R atlas:atlas /app /data
COPY --from=build --chown=atlas:atlas /out/atlas-core /app/atlas-core
COPY --chown=atlas:atlas command-catalog /app/command-catalog
COPY --chown=atlas:atlas sighting-catalog /app/sighting-catalog
COPY --chown=atlas:atlas docker/entrypoint.sh /usr/local/bin/atlas-core-entrypoint
RUN chmod +x /usr/local/bin/atlas-core-entrypoint
ENV ATLAS_CORE_HOST=0.0.0.0 \
    ATLAS_CORE_PORT=8080 \
    ATLAS_CORE_COMMAND_CATALOG_PATH=/app/command-catalog/catalog.json \
    ATLAS_CORE_SIGHTING_CATALOG_PATH=/app/sighting-catalog/catalog.json \
    ATLAS_CORE_OBJECT_STORAGE_ROOT=/data/object-storage
EXPOSE 8080
# The container starts as root only to run the entrypoint, which fixes
# ownership on the persistent /data/object-storage volume (Docker mounts a
# fresh named volume as root:root by default) and then drops to UID 10001
# via su-exec before exec'ing the Go binary. After that point the service
# runs entirely as the unprivileged atlas user.
# Further hardening (optional for ops): read-only root filesystem, drop Linux
# capabilities, or use a separate init container for volume chown.
ENTRYPOINT ["/usr/local/bin/atlas-core-entrypoint"]
CMD ["/app/atlas-core"]
