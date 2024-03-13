.PHONY: test format vet staticcheck magneticod magneticow

all: test magneticod magneticow

magneticod:
	go install --tags fts5 ./cmd/magneticod

magneticow:
	go install --tags fts5 ./cmd/magneticow

vet:
	go vet ./...

test:
	CGO_ENABLED=1 go test --tags fts5 -v -race ./...

format:
	gofmt -w ./cmd/
	gofmt -w ./dht/
	gofmt -w ./metadata/
	gofmt -w ./persistence/
	gofmt -w ./util/
	gci write -s standard -s default -s "prefix(github.com/tgragnato/magnetico)" -s blank -s dot .
