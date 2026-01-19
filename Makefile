.PHONY: test test-verbose test-coverage test-race test-short clean build run

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage report
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detector
test-race:
	go test -race ./...

# Run only short tests (useful for quick feedback)
test-short:
	go test -short ./...

# Run tests for specific package
test-services:
	go test -v ./internal/services/...

test-handlers:
	go test -v ./internal/handlers/...

test-middleware:
	go test -v ./internal/middleware/...

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# Build the application
build:
	go build -o bin/poc-finance ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run tests and show coverage percentage
coverage:
	@go test -coverprofile=coverage.out ./... 2>/dev/null
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@rm -f coverage.out

# Run all checks (tests, race detection)
check: test-race
	@echo "All checks passed!"
