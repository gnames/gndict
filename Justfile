# Install just: https://github.com/casey/just
APP := "gndict"
ORG := "github.com/gnames/"

# Project variables
VERSION := `git describe --tags 2>/dev/null || echo "dev"`
DATE := `date -u '+%Y-%m-%d_%H:%M:%S'`

NO_C := "CGO_ENABLED=0"
FLAGS_LD := "-trimpath -ldflags '-w -s -X " + ORG + APP + \
            "/pkg.Build=" + DATE + " -X " + ORG + APP + \
            "/pkg.Version=" + VERSION + "'"
FLAGS_REL := "-trimpath -ldflags '-s -w -X " + ORG + APP + \
             "/pkg.Build=" + DATE + "'"

# Default recipe - install to ~/go/bin
default: install

# Run all tests
test: deps install
    go test -shuffle=on -count=1 -race -coverprofile=coverage.txt -covermode=atomic ./...

# Download dependencies
deps:
    go mod download

# Build the binary (development build with timestamp and git version)
build:
    @mkdir -p bin
    {{NO_C}} go build {{FLAGS_LD}} -o bin/{{APP}}
    @echo "✅ {{APP}} built to bin/{{APP}}"

# Build release binary (uses version.go for Version, timestamp for Build)
build-release:
    @mkdir -p bin
    {{NO_C}} go build {{FLAGS_REL}} -o bin/{{APP}}
    @echo "✅ {{APP}} release binary built to bin/{{APP}}"

# Install to ~/go/bin (development build with timestamp and git version)
install:
    {{NO_C}} go install {{FLAGS_LD}}
    @echo "✅ {{APP}} installed to ~/go/bin/{{APP}}"

# Clean build artifacts
clean:
    rm -rf bin coverage.txt
    go clean
    @echo "✅ Cleaned build artifacts"

# Format all Go code
fmt:
    go fmt ./...

# Run linter (requires golangci-lint)
lint:
    golangci-lint run

# Tidy dependencies
tidy:
    go mod tidy

# Verify project builds and all tests pass
verify: fmt tidy test build
    @echo "✅ Verification complete: code formatted, dependencies tidied, tests passing, build successful"
