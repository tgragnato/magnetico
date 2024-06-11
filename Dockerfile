FROM golang:alpine3.20 as builder
ENV GOOS=linux
ENV GOARCH=amd64
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
COPY . .
RUN go mod download && go build --tags fts5 . && go build --tags fts5 .

FROM alpine:3.20
WORKDIR /tmp
COPY --from=builder /workspace/magnetico /usr/bin/
RUN echo '#!/bin/sh' >> /usr/bin/magneticod \
    && echo '/usr/bin/magnetico "$@" --daemon' >> /usr/bin/magneticod \
    && chmod +x /usr/bin/magneticod
RUN echo '#!/bin/sh' >> /usr/bin/magneticow \
    && echo '/usr/bin/magnetico "$@" --web' >> /usr/bin/magneticow \
    && chmod +x /usr/bin/magneticow
ENTRYPOINT ["/usr/bin/magnetico"]
LABEL org.opencontainers.image.source=https://github.com/tgragnato/magnetico
