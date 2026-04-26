FROM golang:1.25 AS build
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/atlas-core ./cmd/atlas-core

FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out/atlas-core /app/atlas-core
COPY command-catalog /app/command-catalog
COPY sighting-catalog /app/sighting-catalog
ENV ATLAS_CORE_HOST=0.0.0.0 \
    ATLAS_CORE_PORT=8080 \
    ATLAS_CORE_COMMAND_CATALOG_PATH=/app/command-catalog/catalog.json \
    ATLAS_CORE_SIGHTING_CATALOG_PATH=/app/sighting-catalog/catalog.json \
    ATLAS_CORE_OBJECT_STORAGE_ROOT=/data/object-storage
EXPOSE 8080
ENTRYPOINT ["/app/atlas-core"]
