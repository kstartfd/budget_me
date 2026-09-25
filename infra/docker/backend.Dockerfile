# Production image for the Go backend (cmd/http_server + cmd/migrate + cmd/admin
# + cmd/worker).
# Build context is the repo root (see infra/compose/prod.yml: context ../..),
# so all COPY paths are relative to the repo root (backend/...).
#
# CGO is disabled: no .go source imports a C-backed SQLite driver — the only
# sqlite in go.sum is modernc.org/sqlite (pure Go, pulled transitively by
# goose) and it is never compiled in. A fully static binary lets us ship on
# distroless/static. Migrations are embedded via go:embed (backend/migrations),
# so no .sql files need to be copied into the final image.

# ── builder ────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Warm the module cache first; only re-download when go.mod/go.sum change.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Now the source.
COPY backend/ ./

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/server  ./cmd/http_server \
 && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
 && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/admin   ./cmd/admin \
 && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/worker  ./cmd/worker

# Item photos (internal/photostore) land on local disk, not S3 — pre-create
# the directory here so the COPY below can hand it to distroless already
# owned by uid 65532. That matters because distroless has no shell to mkdir
# or chown at runtime: when prod.yml first mounts a named volume over this
# path, Docker seeds the volume from the image directory it's replacing,
# ownership included — so the volume comes up already writable by nonroot.
RUN mkdir -p /out/data/uploads/items

# ── runtime ────────────────────────────────────────────────────────────────
# distroless/static:nonroot — no shell, no package manager, runs as uid 65532.
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/server  /server
COPY --from=builder /out/migrate /migrate
COPY --from=builder /out/admin   /admin
COPY --from=builder /out/worker  /worker
COPY --from=builder --chown=65532:65532 /out/data /data

USER nonroot
EXPOSE 8080

ENTRYPOINT ["/server"]
