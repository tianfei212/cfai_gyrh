.PHONY: tidy build test run migrate clean

# Go module tidy
tidy:
	go mod tidy

# Build backend
build:
	go build ./backend/...

# Run tests
test:
	go test ./...

# Run development server
run:
	go run ./backend/cmd/server

# Database migration
migrate:
	go run ./backend/cmd/migrate

# Clean build artifacts
clean:
	rm -rf ./backend/data
	rm -rf ./backend/logs
