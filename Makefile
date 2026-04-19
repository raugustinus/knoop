.PHONY: build test run clean install

build:
	CGO_ENABLED=1 go build -tags fts5 -o bin/knoop ./cmd/knoop

test:
	CGO_ENABLED=1 go test -tags fts5 ./...

install:
	CGO_ENABLED=1 go install -tags fts5 ./cmd/knoop

clean:
	rm -rf bin/
