.PHONY: build clean test lint docker-build docker-up docker-down

BINARY := remnawave-limiter

all: build

build:
	@echo "🔨 Сборка..."
	go mod download
	go build -ldflags="-s -w" -o bin/$(BINARY) ./cmd/limiter/
	@echo "✅ Готово!"

clean:
	rm -rf bin/

test:
	go test -v ./...

lint:
	go vet ./...
	go fmt ./...

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down
