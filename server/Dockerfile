# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS build
WORKDIR /src
ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/. ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /out/agent-message-server .

FROM alpine:3.20
RUN apk add --no-cache nodejs su-exec && addgroup -S app && adduser -S -G app app
WORKDIR /app

COPY --from=build /out/agent-message-server /usr/local/bin/agent-message-server
COPY server/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

ENV SERVER_ADDR=:8080 \
    UPLOAD_DIR=/var/lib/agent-message/uploads \
    WEB_PUSH_STATE_DIR=/var/lib/agent-message/web-push
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
