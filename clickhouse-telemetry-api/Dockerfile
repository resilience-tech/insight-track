# syntax=docker/dockerfile:1.7
FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go test ./... \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/clickhouse-telemetry-api ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app -u 10001 app
COPY --from=build /out/clickhouse-telemetry-api /usr/local/bin/clickhouse-telemetry-api
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/clickhouse-telemetry-api"]
