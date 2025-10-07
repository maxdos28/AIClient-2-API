.PHONY: all build run test clean docker

# Variables
BINARY_NAME=aiproxy
DOCKER_IMAGE=go-aiproxy
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-s -w -X main.Version=${VERSION}"

# Default target
all: test build

# Build the application
build:
	go build ${LDFLAGS} -o ${BINARY_NAME} cmd/server/main.go

# Run the application
run: build
	./${BINARY_NAME}

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Lint code
lint:
	golangci-lint run

# Clean build artifacts
clean:
	go clean
	rm -f ${BINARY_NAME}
	rm -f coverage.out coverage.html

# Build Docker image
docker:
	docker build -t ${DOCKER_IMAGE}:${VERSION} .
	docker tag ${DOCKER_IMAGE}:${VERSION} ${DOCKER_IMAGE}:latest

# Run with Docker Compose
docker-up:
	docker-compose up -d

# Stop Docker Compose
docker-down:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f

# Install dependencies
deps:
	go mod download
	go mod tidy

# Update dependencies
update-deps:
	go get -u ./...
	go mod tidy

# Generate mocks (if using mockgen)
generate:
	go generate ./...

# Cross-compilation
build-all:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 cmd/server/main.go
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 cmd/server/main.go
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe cmd/server/main.go

# Create release
release: clean test build-all
	@echo "Release ${VERSION} created in dist/"

# Local development with hot reload (requires air)
dev:
	air

# Security scan
security:
	gosec ./...

# Check for vulnerabilities
vuln-check:
	govulncheck ./...