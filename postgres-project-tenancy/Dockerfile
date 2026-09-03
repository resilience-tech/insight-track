# syntax=docker/dockerfile:1.7
FROM golang:1.27.1-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/project-tenancy ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app -u 10001 app
COPY --from=build /out/project-tenancy /usr/local/bin/project-tenancy
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/project-tenancy"]
