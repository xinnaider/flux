.PHONY: build run test clean docker-build docker-up

BINARY_NAME=loadbalancer
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v -race ./...

clean:
	rm -rf $(BUILD_DIR)

docker-build:
	docker build -t loadbalancer:latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
