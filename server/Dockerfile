# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS build
WORKDIR /src

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/. ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/agent-messenger-server .

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app

COPY --from=build /out/agent-messenger-server /usr/local/bin/agent-messenger-server

ENV SERVER_ADDR=:8080
EXPOSE 8080

USER app
ENTRYPOINT ["/usr/local/bin/agent-messenger-server"]
