# Engineering Standards

This document is the authoritative reference for engineering practices in Magnetico. It complements the project overview in the root README and the operational defaults in the config example with repository-specific rules for the DHT crawler, persistence backends, and web-facing bits.

Magnetico is a Go service that focuses on resilient network discovery, multi-database persistence, and serving searchable torrent metadata. The standards below are therefore aimed at correctness, operability, and graceful degradation under partial failure.

---

## Table of Contents

1. [Scope and intent](#scope-and-intent)
2. [Tooling](#tooling)
3. [Code style](#code-style)
4. [Error handling](#error-handling)
5. [Logging and observability](#logging-and-observability)
6. [Import grouping and package boundaries](#import-grouping-and-package-boundaries)
7. [Context and cancellation](#context-and-cancellation)
8. [Duration, time, and timestamps](#duration-time-and-timestamps)
9. [Persistence and backend contracts](#persistence-and-backend-contracts)
10. [Network and DHT reliability](#network-and-dht-reliability)
11. [Concurrency and goroutine ownership](#concurrency-and-goroutine-ownership)
12. [Testing](#testing)
13. [Dependency minimisation](#dependency-minimisation)
14. [Clean code and maintenance](#clean-code-and-maintenance)

---

## Scope and intent

The repository is organised around a few clear responsibilities:

- DHT crawling and discovery logic in the `dht` and `metadata` packages
- persistence abstraction and backend-specific implementations in `persistence`
- web and API serving in `web`
- CLI flag parsing and runtime assembly in `opflags` and `main.go`
- low-level protocols and encoders such as `bencode`, `metainfo`, and `types`

Engineering decisions should preserve these boundaries. Do not move networking, parsing, or persistence concerns into unrelated packages only to reduce a few lines of code.

The design philosophy is intentionally conservative:

- prefer correctness over cleverness
- degrade gracefully when an upstream peer, network node, or database backend misbehaves
- keep interfaces simple and explicit
- keep the process robust enough to run as a long-lived crawler without operator intervention

---

## Tooling

Run before every commit:

```sh
goimports -w .               # format + fix import grouping (subsumes gofmt)
go vet ./...                 # static analysis
go test ./...                # unit testing
```

Run periodically:

```sh
go test -race ./...          # race detector — mandatory, not optional 
go mod tidy                  # keep go.mod/go.sum clean
golangci-lint run            # extended linting (if installed)
```

In this repository, the primary contract is the Go compiler and the test suite. We do not maintain a larger custom toolchain for lint, static analysis, or CI-only conventions beyond what is already standard in the Go ecosystem.

Rules:

- do not add tooling just to solve tiny style disagreements
- keep new dependencies minimal and justified
- prefer the standard library where it covers the use case cleanly
- if a feature needs a backend driver or protocol library, place it behind the existing persistence or protocol abstractions
- library code must never call log.Fatal, os.Exit, or panic; return an error or degrade gracefully instead

---

## Code style

### Naming and intent

Prefer names that describe behaviour and domain meaning over implementation details.

```go
func (d *Database) DoesTorrentExist(infoHash []byte) (bool, error)
```

is clearer than a vague name that hides the actual contract.

### Comments

Default to no comments. Add one only when the reason is not obvious from the code itself, such as a workaround for a peer protocol quirk, a tradeoff in a DHT edge case, or a compatibility requirement in persistence logic.

Never write comments that merely restate the function name or the obvious control flow.

### Small functions and narrow responsibilities

This codebase benefits from clear separation of concerns: parser, DHT manager, persistence layer, and HTTP handlers should not become giant monoliths. A function that does four unrelated kinds of work should usually be split.

Keep state transitions explicit, and keep side effects local whenever possible.

---

## Error handling

Errors are part of the API contract. Do not hide them or convert them into meaningless success states.

Wrap errors with context:

```go
return fmt.Errorf("persistence: add torrent: %w", err)
```

Do not do this:

```go
return fmt.Errorf("persistence: add torrent: %v", err)
```

The latter loses the chain and breaks `errors.Is` / `errors.As` checks.

Project-specific rules:

- library code should not call `log.Fatal`, `os.Exit`, or `panic` for recoverable conditions
- the boundary at `main.go` may terminate the process when a startup or operational dependency is unavailable
- when a database, network, or filesystem call fails, the relevant layer should return an error and let a higher layer decide whether to continue, degrade, or abort
- use explicit validation at system boundaries, not deep inside pure logic that already assumes well-formed data

---

## Logging and observability

The repository currently uses the standard library `log` package sparsely. Keep that pattern unless a project-wide decision adopts a heavier logger.

Preferred patterns:

```go
log.Printf("magnetico is ready to serve on %s\n", address)
log.Printf("Could not close database! %s\n", err.Error())
```

Avoid:

- formatting values into a single opaque string when structured data would be clearer
- logging the same event at every retry with no useful differentiation
- logging sensitive data or raw credentials when not strictly necessary

Operational guidance:

- log startup, shutdown, and configuration problems at a high level
- log network or database errors with enough context to diagnose a failing backend
- do not emit excessive per-peer logs in hot DHT loops; prefer bounded diagnostics and counters
- use `log.Printf` for lifecycle and failure signals, not for deep algorithm trace output

---

## Import grouping and package boundaries

Keep imports in three groups, with a blank line between each block:

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/goccy/go-yaml"

    "tgragnato.it/magnetico/v2/persistence"
    "tgragnato.it/magnetico/v2/web"
)
```

This repository does not need a more elaborate policy than that. The important part is that imports remain readable and that package boundaries are respected.

Rules:

- keep `main`, `opflags`, `persistence`, `dht`, `web`, and protocol packages independent
- do not import a higher-level package from a lower-level utility package without a clear need
- avoid circular dependencies; if a package needs a backend abstraction, depend on the interface rather than a concrete implementation

---

## Context and cancellation

Where the code may block or perform I/O, prefer a `context.Context` flow through the API.

Examples from this repository that already follow this principle are the HTTP server and database access paths. The general rule is:

- accept a context when the function may block on I/O, a database query, or a network operation
- propagate it through the call chain instead of storing it in a struct field
- make long-lived goroutines cancelable, especially when they are started by a parent service or a CLI lifecycle

The standard library `net/http` and database interfaces already encourage this pattern. Keep it consistent.

---

## Duration, time, and timestamps

This project works with network events, database timestamps, and crawler lifecycles; it is important to keep time handling explicit.

Rules:

- use UTC or Unix timestamps consistently in storage and logic when the data is not inherently local
- avoid converting to local time in core logic where the system is meant to behave deterministically
- prefer explicit durations for crawler deadlines, timeouts, and backoff rather than implicit wall-clock assumptions
- if a timestamp is imported from an external source, normalise it before it is used for comparisons or ordering

In a DHT crawler, time is operational data; never treat it as an afterthought.

---

## Persistence and backend contracts

Magnetico intentionally supports multiple persistence backends: SQLite, PostgreSQL/CockroachDB, ZeroMQ, RabbitMQ, and Bitmagnet. This is a real project constraint, not an implementation detail to ignore.

The interface in `persistence/interface.go` is the central compatibility contract. When changing the persistence layer:

- keep the interface stable unless a real dependency change requires it
- keep backend-specific logic inside the concrete implementation
- do not leak SQLite or PostgreSQL details into the DHT or web layers
- preserve the same semantics across backends when they are expected to behave identically

Rules for new storage-related code:

- keep SQL or transport logic close to the relevant backend file
- return errors with enough context to diagnose the failing backend
- do not silently swallow a database or transport failure
- ensure serialisation or export/import flows remain reversible and testable

---

## Network and DHT reliability

The DHT code path must behave correctly under churn, timeouts, incomplete peers, and stale metadata. The project explicitly expects partial failures to be normal.

Engineering rules for this domain:

- treat a single peer failure as non-fatal
- keep node and peer selection bounded, avoid unbounded memory growth
- use timeout-based logic around network calls and discovery retries
- keep the crawler operational even when some peers are slow or misbehaving
- when a result is duplicate, stale, or structurally invalid, drop it without crashing the broader loop

This is one of the primary values of the project: the architecture must tolerate a noisy and adversarial network rather than assume perfect conditions.

---

## Concurrency and goroutine ownership

This codebase uses concurrency in the DHT manager, web startup, and persistence flows. Concurrency is useful, but it must remain comprehensible and bounded.

Rules:

- if a goroutine is owned by a parent function or service, it must be shut down by that owner
- use channels or cancellation to stop workers cleanly instead of relying on background goroutines to self-resolve in an ad-hoc way
- do not start unbounded goroutines in loops without a clear termination strategy
- protect shared mutable state with mutexes or channel ownership, not with “best effort” locking scattered across the code

The DHT manager pattern in `dht/managers.go` is a good example: a result channel is buffered and swapped under a lock when the pipeline is under pressure, rather than letting a single blocked consumer corrupt the whole flow.

---

## Testing

The project already uses a broad `t.Parallel()` pattern in tests, and that is appropriate for independent unit tests. Preserve it when safe.

General rules:

- call `t.Parallel()` as the first statement in independent tests
- do not use `t.Parallel()` in tests that mutate package-level state or shared test fixtures
- prefer table-driven tests for repeated input variations
- use `t.Log` for auxiliary diagnostics, not `fmt.Print` or `fmt.Printf`
- keep tests focused on real behaviour instead of internal implementation details

For database-backed code, use the real package contract rather than overspecifying mocks. The repository already includes `DATA-DOG/go-sqlmock`, which is the right tool when testing SQL behaviour with strict expectations.

For persistence and network-heavy code, the most important checks are:

- can the function fail correctly?
- does it keep the caller contract intact?
- does it degrade gracefully under duplicate or partial data?

---

## Dependency minimisation

The project dependencies in `go.mod` are intentionally pragmatic. They cover real needs such as the PostgreSQL driver (`pgx`), compression, DHT metadata handling, and the HTTP stack.

The general rule is:

- prefer the standard library for common data structures, encoding, crypto, and network primitives
- add third-party packages only when the repository truly needs them for protocol compatibility, database drivers, or specific runtime features
- check that a dependency is actively maintained and compatible with the project’s licensing model
- do not add a dependency for a trivial helper that can be implemented in a few lines of Go

Current clearly justified dependencies include:

- `github.com/jackc/pgx/v5` for PostgreSQL connectivity
- `github.com/mattn/go-sqlite3` for embedded SQLite
- `github.com/prometheus/client_golang` for metrics exposure
- `github.com/rabbitmq/amqp091-go` and `github.com/pebbe/zmq4` for message-bus integrations
- `github.com/goccy/go-yaml` for config parsing

---

## Clean code and maintenance

Do not leave dead code in place. If a path is no longer needed, remove it rather than leaving behind a dormant implementation.

Rules:

- no commented-out code blocks
- no `TODO` or `FIXME` markers in committed code
- exported functions and types should be documented when they are part of a public package contract
- if a discard value is intentionally ignored, make the reason obvious
- keep package-level responsibilities narrow and discoverable

The repository benefits from a direct style: compact, explicit, and context-aware code is preferable to clever abstractions that hide the actual failure modes.

---

## Practical summary

The most important engineering constraints for Magnetico are these:

1. preserve the DHT, persistence, and web boundaries
2. fail with context, not silence
3. keep long-lived work cancelable and bounded
4. prefer robust degradation over heroic single-point recovery
5. keep tests and runtime behavior honest across the supported storage backends

This is a project about surviving a hostile and dynamic network while serving searchable metadata reliably. The engineering standards above exist to keep that system predictable and maintainable as it grows.
