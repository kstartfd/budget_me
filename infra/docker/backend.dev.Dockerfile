FROM golang:1.25-alpine

RUN apk add --no-cache git

# Hot reload tool
RUN go install github.com/air-verse/air@latest

WORKDIR /app

# Pre-warm module cache when go.mod changes (mod cache is preserved via named volume)
COPY backend/go.mod ./
RUN go mod download || true

EXPOSE 8080

# go mod tidy: creates/updates go.sum on the mounted host filesystem on first run
# air: rebuilds and restarts on file changes
CMD ["sh", "-c", "go mod tidy && air -c .air.toml"]
