default:
	@just --list

# Clean build artifacts and test outputs
clean:
	rm -rf dist/
	rm -rf build/
	go clean

# Build snapshot release
snapshot: clean
    goreleaser release --snapshot --clean

# Build full release
release version: clean
    goreleaser release --clean
	GOPROXY=proxy.golang.org go list -m github.com/suderio/dndsl@{{version}}

# Build the binary locally
build:
	go build -o dndsl ./main.go

# Run all tests (after cleaning)
test: clean
    go test ./...

# Show test coverage in the terminal
coverage:
	go test -cover ./...

