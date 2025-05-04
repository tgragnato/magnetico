FROM golang:alpine3.21 AS builder
ENV CGO_ENABLED=1
ENV CGO_CFLAGS=-D_LARGEFILE64_SOURCE
ENV CC=clang
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
COPY . .
RUN apk add --no-cache clang lld libsodium-dev zeromq-dev && go mod download && go build --tags fts5 .
RUN ln -s magnetico magneticod
RUN ln -s magnetico magneticow

FROM alpine:3.21
WORKDIR /tmp
COPY --from=builder /workspace/magnetico /usr/bin/
COPY --from=builder /workspace/magneticod /usr/bin/magneticod
COPY --from=builder /workspace/magneticow /usr/bin/magneticow
RUN apk add --no-cache libsodium libzmq
ENTRYPOINT ["/usr/bin/magnetico"]
LABEL org.opencontainers.image.title="magnetico"
LABEL org.opencontainers.image.description="Autonomous (self-hosted) BitTorrent DHT search engine"
LABEL org.opencontainers.image.url="https://tgragnato.it/magnetico/"
LABEL org.opencontainers.image.source="https://github.com/tgragnato/magnetico"
LABEL org.opencontainers.image.licenses="AGPL-3.0"
LABEL io.containers.autoupdate=registry
