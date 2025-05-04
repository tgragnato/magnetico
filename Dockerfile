FROM golang:alpine3.21 AS builder
ENV CGO_ENABLED=1
ENV CGO_CFLAGS=-D_LARGEFILE64_SOURCE
ENV CGO_LDFLAGS='-fuse-ld=lld -static -lstdc++ -lsodium -lzmq'
ENV CC=clang
RUN apk add --no-cache clang lld libc-dev musl-dev libstdc++ libsodium-dev libsodium-static zeromq-dev libzmq-static
WORKDIR /workspace/bin
RUN ln -s magnetico magneticod
RUN ln -s magnetico magneticow
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build --tags fts5 -ldflags='-linkmode=external' -o bin/magnetico .

FROM ghcr.io/anchore/syft:latest AS sbomgen
COPY --from=builder /workspace/bin/magnetico /usr/bin/magnetico
RUN ["/syft", "--output", "spdx-json=/magnetico.spdx.json", "/usr/bin/magnetico"]

FROM cgr.dev/chainguard/static:latest
WORKDIR /tmp
COPY --from=builder /workspace/bin /usr/bin
COPY --from=sbomgen /magnetico.spdx.json /var/lib/db/sbom/magnetico.spdx.json
ENTRYPOINT ["/usr/bin/magnetico"]
LABEL org.opencontainers.image.title="magnetico"
LABEL org.opencontainers.image.description="Autonomous (self-hosted) BitTorrent DHT search engine"
LABEL org.opencontainers.image.url="https://tgragnato.it/magnetico/"
LABEL org.opencontainers.image.source="https://github.com/tgragnato/magnetico"
LABEL org.opencontainers.image.licenses="AGPL-3.0"
LABEL io.containers.autoupdate=registry
