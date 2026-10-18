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

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/server /server
COPY --from=build /out/migrate /migrate
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
