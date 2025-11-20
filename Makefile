.PHONY: build clean test example

# Build the CLI
build:
	go build -o gigsheets

# Clean build artifacts
clean:
	rm -f gigsheets *.pdf

# Run tests
test:
	go test ./...

# Generate example PDF
example: build
	./gigsheets generate --config example/config.yaml --gig example/gig.yaml

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Build for multiple platforms
build-all: build-linux build-macos build-windows

build-linux:
	GOOS=linux GOARCH=amd64 go build -o gigsheets-linux-amd64

build-macos:
	GOOS=darwin GOARCH=amd64 go build -o gigsheets-darwin-amd64

build-windows:
	GOOS=windows GOARCH=amd64 go build -o gigsheets-windows-amd64.exe