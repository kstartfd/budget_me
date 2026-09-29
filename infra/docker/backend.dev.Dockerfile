FROM golang:1.25-alpine

RUN apk add --no-cache git

# Hot reload tool. Pinned, not @latest (see CLAUDE.md / project convention) —
# air v1.67.4 bumped its go.mod to `go 1.26`, which this image's Go 1.25
# can't build; v1.67.1 is the newest release still on go1.25.
RUN go install github.com/air-verse/air@v1.67.1

WORKDIR /app

# Pre-warm module cache when go.mod changes (mod cache is preserved via named volume)
COPY backend/go.mod ./
RUN go mod download || true

EXPOSE 8080

# go mod tidy: creates/updates go.sum on the mounted host filesystem on first run
# air: rebuilds and restarts on file changes
CMD ["sh", "-c", "go mod tidy && air -c .air.toml"]
