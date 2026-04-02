.PHONY: build clean install test deploy

GO_VERSION := 1.26

LIMITER_BIN := remnawave-limiter

INSTALL_PATH := /usr/local/bin

all: build

build:
	@echo "🔨 Сборка Remnawave Limiter..."
	go mod download
	go build -ldflags="-s -w" -o bin/$(LIMITER_BIN) ./cmd/limiter
	@echo "✅ Сборка завершена!"

install: build
	@echo "📦 Установка бинарников..."
	sudo cp bin/$(LIMITER_BIN) $(INSTALL_PATH)/
	sudo chmod +x $(INSTALL_PATH)/$(LIMITER_BIN)
	@echo "✅ Установка завершена!"

clean:
	@echo "🗑️  Очистка..."
	rm -rf bin/
	@echo "✅ Очистка завершена!"

test:
	@echo "🧪 Запуск тестов..."
	go test -v ./...

deploy: build
	@echo "📦 Установка бинарников..."
	sudo cp bin/$(LIMITER_BIN) $(INSTALL_PATH)/
	sudo chmod +x $(INSTALL_PATH)/$(LIMITER_BIN)
	@echo "🗑️  Очистка исходников и кэша..."
	go clean -modcache -cache
	rm -rf bin/
	@echo "✅ Деплой завершён, исходники очищены!"

lint:
	@echo "🔍 Проверка кода..."
	go vet ./...
	go fmt ./...
