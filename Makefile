.PHONY: help run build test vet tidy clean up down logs ps

BINARY_NAME=url-short
BIN_DIR=bin
CMD_DIR=./cmd/api

help: ## Показать все доступные команды
	@echo "📋 Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Собрать бинарный файл
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✅ Бинарник: $(BIN_DIR)/$(BINARY_NAME)"

run: ## Запустить приложение (без сборки)
	go run $(CMD_DIR)

test: ## Запустить все тесты (с детектором гонок)
	go test -race ./...

vet: ## Статический анализ
	go vet ./...

tidy: ## Обновить зависимости
	go mod tidy

clean: ## Очистить собранные файлы
	rm -rf $(BIN_DIR)
	go clean -cache

up: ## Запустить все сервисы через Docker Compose
	docker compose -f docker-compose.yaml up -d --build
	@echo "✅ Сервисы запущены! Проверьте: curl http://localhost:8080/health"

down: ## Остановить и удалить все сервисы
	docker compose -f docker-compose.yaml down
	@echo "✅ Контейнеры остановлены"

logs: ## Показать логи
	docker compose -f docker-compose.yaml logs -f

ps: ## Показать статус контейнеров
	docker compose -f docker-compose.yaml ps