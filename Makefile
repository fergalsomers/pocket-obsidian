APP_NAME = pocket-obsidian

.PHONY: all build test clean download

all: download build

download:
	go mod tidy
	go mod download

build: vet test
	go build -o $(APP_NAME) .

test: vet
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

vet: 
	go vet ./...

clean:
	rm -f $(APP_NAME)