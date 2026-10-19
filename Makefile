.PHONY: build test vet check-leaks clean

build:
	go build -o dist/reitingtuvas ./cmd/reitingtuvas

test:
	go test ./...

vet:
	go vet ./...

check-leaks:
	go run ./internal/tools/leakcheck

clean:
	rm -rf dist/
