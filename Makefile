# Makefile — sql-cli
# Atalhos de build, teste e instalação para o projeto.

BINARY_NAME  := sql-cli
BUILD_DIR    := bin
CMD_PATH     := ./cmd/sql-cli
MODULE       := github.com/stanley/sql-cli

# Versão extraída via git tag (fallback para "dev").
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS      := -s -w -X main.version=$(VERSION)

.PHONY: all build run install clean test vet lint tidy

## all: Compila o binário (padrão).
all: build

## build: Compila o binário em ./bin/sql-cli
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "✓ Binário gerado: $(BUILD_DIR)/$(BINARY_NAME)"

## run: Compila e executa diretamente.
run:
	go run $(CMD_PATH)

## install: Instala o binário no $GOPATH/bin (ou $GOBIN).
install:
	go install -ldflags "$(LDFLAGS)" $(CMD_PATH)
	@echo "✓ Instalado em $$(go env GOPATH)/bin/$(BINARY_NAME)"

## test: Executa todos os testes com cobertura.
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Relatório de cobertura: coverage.html"

## vet: Executa o go vet para verificar erros estáticos.
vet:
	go vet ./...

## lint: Executa o golangci-lint (deve estar instalado).
lint:
	golangci-lint run ./...

## tidy: Limpa e organiza o go.mod e go.sum.
tidy:
	go mod tidy

## clean: Remove os artefatos de build.
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "✓ Limpo."

## help: Lista os alvos disponíveis.
help:
	@grep -E '^##' Makefile | sed 's/## //'
