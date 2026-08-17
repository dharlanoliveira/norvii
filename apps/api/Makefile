.PHONY: build ci coverage dependencies-check format format-check migrate migration-status test test-race verify-persistence vet

build:
	go build ./...

coverage:
	go test -covermode=atomic -coverprofile=coverage.out ./...

dependencies-check:
	go mod tidy -diff
	go mod verify

format:
	gofmt -w $$(find . -name '*.go' -type f)

format-check:
	test -z "$$(gofmt -l $$(find . -name '*.go' -type f))"

migrate:
	go run ./cmd/migrate

migration-status:
	go run ./cmd/migration-status

test:
	go test ./...

test-race:
	go test -race ./...

verify-persistence:
	go run ./cmd/verify-persistence

vet:
	go vet ./...

ci: dependencies-check format-check vet test-race build
