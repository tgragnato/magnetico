# magnetico
*Autonomous (self-hosted) BitTorrent DHT search engine suite.*

[![Go](https://github.com/tgragnato/magnetico/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/tgragnato/magnetico/actions/workflows/go.yml)
[![CodeQL](https://github.com/tgragnato/magnetico/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/tgragnato/magnetico/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tgragnato/magnetico)](https://goreportcard.com/report/github.com/tgragnato/magnetico)
[![codecov](https://codecov.io/gh/tgragnato/magnetico/branch/main/graph/badge.svg)](https://codecov.io/gh/tgragnato/magnetico)

![Flow of Operations](/doc/operations.svg)

magnetico is the first autonomous (self-hosted) BitTorrent DHT search engine suite that is *designed for end-users*. The suite consists of a single binary with two components:

- a crawler for the BitTorrent DHT network, which discovers info hashes and fetches metadata from the peers.
- a lightweight web interface for searching and browsing the torrents discovered by its counterpart.

This allows anyone with a decent Internet connection to access the vast amount of torrents waiting to be discovered within the BitTorrent DHT space, *without relying on any central entity*.

**magnetico** liberates BitTorrent from the yoke of centralised trackers & web-sites and makes it
*truly decentralised*. Finally!

## Easy Run and Compilation

The easiest way to run magnetico is to use the OCI image built within the CI pipeline:
- `docker pull ghcr.io/tgragnato/magnetico:latest`
- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest --help`
- `docker run --rm -it -v <your_data_dir>:/data -p 8080:8080/tcp ghcr.io/tgragnato/magnetico:latest --addr=0.0.0.0:8080 --database=sqlite3:///data/magnetico.sqlite3`
- visit `http://localhost:8080`

To compile using the standard Golang toolchain:
- Download the latest golang release from [the official website](https://go.dev/dl/)
- Follow the [installation instructions for your platform](https://go.dev/doc/install)
- Checkout the repository and run `go install --tags fts5 .`
- The `magnetico` binary is now available in your `$GOBIN` directory

### PostgreSQL

PostgreSQL is a powerful, scalable database with advanced features for complex applications and high concurrency.
SQLite is lightweight and easy to embed, ideal for simpler or smaller-scale applications.
You might prefer PostgreSQL if you need scalability, advanced features, and robust concurrency management.

The installation of PostgreSQL varies depending on the OS and the final configuration you want to achieve.
After setting it up, you should create a user, set a password, create a database owned by that user, and load the `pg_trgm` extension.

- `CREATE USER magnetico WITH PASSWORD 'magnetico';`
- `CREATE DATABASE magnetico OWNER magnetico;`
- `\c magnetico`
- `CREATE EXTENSION IF NOT EXISTS pg_trgm;`
- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest --help`
- `docker run --rm -it -p 8080:8080/tcp ghcr.io/tgragnato/magnetico:latest --addr=0.0.0.0:8080 --database=postgres://magnetico:magnetico@localhost:5432/magnetico?sslmode=disable`
- visit `http://localhost:8080`

### CockroachDB

CockroachDB is ideal for situations when horizontal scalability, high availability, and global distribution is required.
It currently does not support the `pg_trgm` extension, which provides functions for trigram-based similarity searching.

- create a user and it's database
- download the TLS certificate of your cluster
- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest --help`
- `docker run --rm -it -v <your_cert_dir>:/data -p 8080:8080/tcp ghcr.io/tgragnato/magnetico:latest --addr=0.0.0.0:8080 --database=cockroach://magneticouser:magneticopass@mycluster.crdb.io:26257/magnetico?sslmode=verify-full&sslrootcert=/data/cc-ca.crt`
- visit `http://localhost:8080`

### ZeroMQ

ZeroMQ is a high-performance messaging library that provides a set of tools for communication between distributed applications.
The integration is designed in the persistence layer as a ZMQ PUB firehose, and works under the zeromq and zmq URL schemas.

- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest --help`
- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest -d --database=zeromq://localhost:5555`

### RabbitMQ

RabbitMQ is an open-source message broker that facilitates communication between distributed systems by queuing and routing messages. It supports multiple messaging protocols and ensures reliable message delivery for scalable applications.
The integration is designed in the persistence layer following the Publish/Subscribe model, and operates over a durable queue named `magnetico` routed through an exchange.

- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest --help`
- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest -d --database=amqp://localhost:5672`

### Bitmagnet

Bitmagnet is a self-hosted BitTorrent indexer, DHT crawler, content classifier and torrent search engine with web UI, GraphQL API and Servarr stack integration.
The integration is designed in the persistence layer as a producer for the import API endpoint, and works under the bitmagnet and bitmagnets URL schemas.

- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest --help`
- `docker run --rm -it ghcr.io/tgragnato/magnetico:latest -d --database=bitmagnet://localhost:3333/import`

## Features

Easy installation & minimal requirements:
  - Easy to build golang static binaries.
  - Root access is *not* required to install or to use.

**magnetico** trawls the BitTorrent DHT by "going" from one node to another, and fetches the metadata using the nodes without using trackers. No reliance on any centralised entity!

Unlike client-server model that web applications use, P2P networks are *chaotic* and **magnetico** is designed to handle all the operational errors accordingly.

High performance implementation in Go: **magnetico** utilizes every bit of your resources to discover as many infohashes & metadata as possible.

**magnetico** features a lightweight web interface to help you access the database without getting on your way.

If you'd like to password-protect the access to **magnetico**, you need to store the credentials
in file. The `credentials` file must consist of lines of the following format: `<USERNAME>:<BCRYPT HASH>`.

- `<USERNAME>` must start with a small-case (`[a-z]`) ASCII character, might contain non-consecutive underscores except at the end, and consists of small-case a-z characters and digits 0-9.
- `<BCRYPT HASH>` is the output of the well-known bcrypt function.

You can use `htpasswd` (part of `apache2-utils` on Ubuntu) to create lines:

```
$  htpasswd -bnBC 12 "USERNAME" "PASSWORD"
USERNAME:$2y$12$YE01LZ8jrbQbx6c0s2hdZO71dSjn2p/O9XsYJpz.5968yCysUgiaG
```

### Screenshots

| ![The Homepage](/doc/homepage.png) | ![Searching for torrents](/doc/search.png) | ![Search result](/doc/result.png) |
|:-------------------------------------------------------------------------------------------------------------------------------------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------:|:---------------------------------------------------------------------------------------------------------------------------------------------:|
|                                                                     __The Homepage__                                                                    |                                                                     __Searching for torrents__                                                                    |                                                     __Viewing the metadata of a torrent__                                                     |

## Why?

BitTorrent, being a distributed P2P file sharing protocol, has long suffered because of the
centralised entities that people depended on for searching torrents (websites) and for discovering
other peers (trackers). Introduction of DHT (distributed hash table) eliminated the need for
trackers, allowing peers to discover each other through other peers and to fetch metadata from the
leechers & seeders in the network. **magnetico** is the finishing move that allows users to search
for torrents in the network, hence removing the need for centralised torrent websites.
