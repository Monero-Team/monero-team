.PHONY: build run test vet fmt fmt-check check tidy

build:
	go build ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Not gofmt-clean:"; echo "$$unformatted"; exit 1; \
	fi

# Run everything CI runs.
check: fmt-check vet test
	./scripts/check-no-external-origins.sh

tidy:
	go mod tidy
