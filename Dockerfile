FROM golang:alpine3.20 as builder
ENV CGO_ENABLED=1
ENV CGO_CFLAGS=-D_LARGEFILE64_SOURCE
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
COPY . .
RUN apk add --no-cache alpine-sdk && go mod download && go build --tags fts5 . && go build --tags fts5 .

FROM alpine:3.20
WORKDIR /tmp
COPY --from=builder /workspace/magnetico /usr/bin/
RUN apk add --no-cache libstdc++ libgcc \
    && echo '#!/bin/sh' >> /usr/bin/magneticod \
    && echo '/usr/bin/magnetico "$@" --daemon' >> /usr/bin/magneticod \
    && chmod +x /usr/bin/magneticod \
    && echo '#!/bin/sh' >> /usr/bin/magneticow \
    && echo '/usr/bin/magnetico "$@" --web' >> /usr/bin/magneticow \
    && chmod +x /usr/bin/magneticow
ENTRYPOINT ["/usr/bin/magnetico"]
LABEL org.opencontainers.image.source=https://github.com/tgragnato/magnetico
