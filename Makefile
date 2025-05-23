.PHONY: build test clean install run-example help

# Переменные
BINARY_NAME=replacer
BUILD_DIR=build

help: ## Показать эту справку
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Собрать бинарный файл
	@echo "🔨 Сборка $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Запустить тесты
	@echo "🧪 Запуск тестов..."
	go test -v ./...

clean: ## Очистить сборочные файлы
	@echo "🧹 Очистка..."
	rm -rf $(BUILD_DIR)
	go clean

install: build ## Установить в GOPATH/bin
	@echo "📦 Установка..."
	go install .
	@echo "✅ Установлено в GOPATH/bin"

run-example: ## Запустить пример с тестовыми данными
	@echo "🚀 Пример работы с тестовыми данными..."
	go run . testdata/source.go testdata/target.go
	@echo "📄 Результат записан в testdata/target.go"

deps: ## Обновить зависимости
	@echo "📚 Обновление зависимостей..."
	go mod tidy
	go mod download

lint: ## Проверить код линтером (требует golangci-lint)
	@echo "🔍 Проверка кода..."
	golangci-lint run

fmt: ## Форматировать код
	@echo "✨ Форматирование кода..."
	go fmt ./...

vet: ## Проверить код vet'ом
	@echo "🔬 Анализ кода..."
	go vet ./...

check: fmt vet test ## Полная проверка кода
