.PHONY: gen build run test clean docker-build

# Generate Goa code
gen:
	go install goa.design/goa/v3/cmd/goa@latest
	goa gen springstreet/api/design
	goa example springstreet/api/design

# Build application
build: gen
	go build -o springstreet-api cmd/api/main.go

# Run application
run:
	go run cmd/api/main.go

# Run tests
test:
	go test ./...

# Clean generated files
clean:
	rm -rf gen/
	rm -f springstreet

# Docker build
docker-build:
	docker build -t springstreet-go:latest .

# Install dependencies
deps:
	go mod download
	go mod tidy

