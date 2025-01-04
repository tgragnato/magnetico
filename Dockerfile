FROM golang:alpine3.21 AS builder
ENV CGO_ENABLED=1
ENV CGO_CFLAGS=-D_LARGEFILE64_SOURCE
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
COPY . .
RUN apk add --no-cache alpine-sdk libsodium-dev zeromq-dev czmq-dev && go mod download && go build --tags fts5 . && go build --tags fts5 .

FROM alpine:3.21
WORKDIR /tmp
COPY --from=builder /workspace/magnetico /usr/bin/
RUN apk add --no-cache libstdc++ libgcc libsodium libzmq czmq \
    && echo '#!/bin/sh' >> /usr/bin/magneticod \
    && echo '/usr/bin/magnetico "$@" --daemon' >> /usr/bin/magneticod \
    && chmod +x /usr/bin/magneticod \
    && echo '#!/bin/sh' >> /usr/bin/magneticow \
    && echo '/usr/bin/magnetico "$@" --web' >> /usr/bin/magneticow \
    && chmod +x /usr/bin/magneticow
ENTRYPOINT ["/usr/bin/magnetico"]
LABEL org.opencontainers.image.title="magnetico"
LABEL org.opencontainers.image.description="Autonomous (self-hosted) BitTorrent DHT search engine"
LABEL org.opencontainers.image.url="https://tgragnato.it/magnetico/"
LABEL org.opencontainers.image.source="https://github.com/tgragnato/magnetico"
LABEL org.opencontainers.image.licenses="AGPL-3.0"
LABEL io.containers.autoupdate=registry
