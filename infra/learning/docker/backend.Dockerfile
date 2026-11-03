# Build context is the REPO ROOT: the backend imports the shared account module
# (libs/account, via the replace in go.mod).
FROM golang:1.25-alpine AS build
WORKDIR /src/apps/learning/backend
COPY libs/account/go.mod libs/account/go.sum /src/libs/account/
COPY apps/learning/backend/go.mod apps/learning/backend/go.sum ./
RUN go mod download
COPY libs/account /src/libs/account
COPY apps/learning/backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/http_server \
 && CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate

# Pictures are read on this machine by tesseract, never sent to an AI agent.
# OCR_LANGS is the language data installed (each is ~20 MB); internal/ocr lists
# the same ones, and a test keeps the two in step. Alpine matches the build
# stage's, so the Go binaries (static) run on it as they did on distroless.
FROM alpine:3.24
ARG OCR_LANGS="bel ces dan deu eng fin fra ita nld nor pol por rus spa swe tur ukr"
RUN apk add --no-cache tesseract-ocr $(for lang in $OCR_LANGS; do printf 'tesseract-ocr-data-%s ' "$lang"; done) \
 && mkdir -p /data/audio \
 && chown -R 65532:65532 /data
COPY --from=build /out/server /server
COPY --from=build /out/migrate /migrate
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]
