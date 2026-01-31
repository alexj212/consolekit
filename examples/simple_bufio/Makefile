.PHONY: build clean run test install

# Build variables
BINARY_NAME=simple-bufio
BUILD_DIR=.

# Build flags
LDFLAGS=-X 'main.Version=dev' \
        -X 'main.BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")' \
        -X 'main.LatestCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")' \
        -X 'main.GitRepo=$(shell git config --get remote.origin.url 2>/dev/null || echo "unknown")' \
        -X 'main.GitBranch=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")'

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Clean complete"

# Run in REPL mode
run: build
	./$(BINARY_NAME)

# Run tests
test:
	@echo "Testing command-line mode..."
	./$(BINARY_NAME) print "Hello World"
	@echo ""
	@echo "Testing script execution..."
	./$(BINARY_NAME) run @test.run
	@echo ""
	@echo "Testing piped input..."
	@printf "print \"Piped test\"\ndate\nexit\n" | ./$(BINARY_NAME)

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install -ldflags "$(LDFLAGS)"
	@echo "Install complete"

# Show version info
version: build
	./$(BINARY_NAME) version

# Help
help:
	@echo "Available targets:"
	@echo "  build    - Build the binary (default)"
	@echo "  clean    - Remove build artifacts"
	@echo "  run      - Build and run in REPL mode"
	@echo "  test     - Run simple tests"
	@echo "  install  - Install to GOPATH/bin"
	@echo "  version  - Show version information"
	@echo "  help     - Show this help message"
