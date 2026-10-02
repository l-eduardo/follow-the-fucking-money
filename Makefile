.PHONY: all build test run clean docker-up docker-down

all: test build

build:
	@mkdir -p bin
	go build -o bin/ftfm ./cmd/ftfm

test:
	go test -v ./...

clean:
	rm -rf bin/

docker-up:
	docker compose -f deployments/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.yml down

run-server: build
	./bin/ftfm server --port 8080
