# ==============================================================================
# Follow-The-Fucking-Money (FTFM) - Makefile
# ==============================================================================
# Utilitário para compilar, executar, gerenciar infraestrutura e testar o FTFM.
# ==============================================================================

SHELL := /bin/bash
PROJECT_NAME := ftfm
COMPOSE_FILE := deployments/docker-compose.yml
COMPOSE_CMD := docker compose -p $(PROJECT_NAME) -f $(COMPOSE_FILE)
BIN_DIR := bin
BIN_NAME := ftfm
BIN_PATH := $(BIN_DIR)/$(BIN_NAME)

# Cores para o terminal
CYAN  := \033[36m
GREEN := \033[32m
YELLOW:= \033[33m
RED   := \033[31m
RESET := \033[0m

.DEFAULT_GOAL := help

##@ Ajuda
.PHONY: help
help: ## Exibe este menu de ajuda interativo
	@echo -e "$(GREEN)==============================================================================$(RESET)"
	@echo -e "$(GREEN)  Follow-The-Fucking-Money (FTFM) - Guia de Comandos$(RESET)"
	@echo -e "$(GREEN)==============================================================================$(RESET)"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUso: make $(CYAN)<comando>$(RESET)\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-18s$(RESET) %s\n", $$1, $$2 } \
		/^##@/ { printf "\n$(YELLOW)%s$(RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""

##@ Desenvolvimento & Compilação
.PHONY: build
build: ## Compila o binário CLI do FTFM em bin/ftfm
	@echo -e "$(CYAN)🔨 Compilando o binário $(BIN_PATH)...$(RESET)"
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-w -s" -o $(BIN_PATH) ./cmd/ftfm
	@echo -e "$(GREEN)✓ Binário gerado com sucesso: $(BIN_PATH)$(RESET)"

.PHONY: test
test: ## Executa todos os testes unitários com cobertura
	@echo -e "$(CYAN)🧪 Executando testes unitários...$(RESET)"
	go test -v -race -cover ./...

.PHONY: clean
clean: ## Remove binários compilados e arquivos temporários
	@echo -e "$(YELLOW)🧹 Limpando binários e arquivos gerados...$(RESET)"
	rm -rf $(BIN_DIR) cover.out coverage.html
	@echo -e "$(GREEN)✓ Limpeza concluída.$(RESET)"

##@ Infraestrutura (Docker Compose)
.PHONY: up
up: ## Sobe toda a stack (Memgraph, Memgraph Lab UI e API Go)
	@echo -e "$(CYAN)🚀 Iniciando toda a stack FTFM em background...$(RESET)"
	$(COMPOSE_CMD) up -d --build
	@echo -e "$(GREEN)✓ Stack em execução!$(RESET)"
	@echo -e "  • Memgraph (Bolt): $(CYAN)bolt://localhost:7687$(RESET)"
	@echo -e "  • Memgraph Lab (UI Web): $(CYAN)http://localhost:3000$(RESET)"
	@echo -e "  • FTFM REST API: $(CYAN)http://localhost:8080$(RESET)"

.PHONY: up-graph
up-graph: ## Sobe apenas o banco Memgraph e o Memgraph Lab (ideal para rodar Go local)
	@echo -e "$(CYAN)🚀 Iniciando Memgraph DB e Memgraph Lab UI...$(RESET)"
	$(COMPOSE_CMD) up -d graphdb graph-ui
	@echo -e "$(GREEN)✓ Banco de grafos e interface Web disponíveis!$(RESET)"
	@echo -e "  • Memgraph (Bolt): $(CYAN)bolt://localhost:7687$(RESET)"
	@echo -e "  • Memgraph Lab (UI Web): $(CYAN)http://localhost:3000$(RESET)"

.PHONY: down
down: ## Para todos os containers do projeto
	@echo -e "$(YELLOW)🛑 Parando containers FTFM...$(RESET)"
	$(COMPOSE_CMD) down
	@echo -e "$(GREEN)✓ Containers parados.$(RESET)"

.PHONY: down-v
down-v: ## Para todos os containers e REMOVE os volumes persistentes do banco
	@echo -e "$(RED)⚠️ Parando containers e apagando volumes de dados...$(RESET)"
	$(COMPOSE_CMD) down -v
	@echo -e "$(GREEN)✓ Containers e volumes removidos.$(RESET)"

.PHONY: logs
logs: ## Exibe os logs contínuos de todos os containers
	$(COMPOSE_CMD) logs -f

.PHONY: ps
ps: ## Lista os containers e status de saúde do FTFM
	$(COMPOSE_CMD) ps

##@ Operações & Aplicação
.PHONY: status
status: build ## Verifica conexão com o Graph DB e exibe contagem de nós
	@echo -e "$(CYAN)🔍 Verificando status do banco de grafos...$(RESET)"
	./$(BIN_PATH) status

.PHONY: init-db
init-db: build ## Aplica constraints UNIQUE e índices no banco de grafos
	@echo -e "$(CYAN)⚙️ Aplicando constraints e índices no Graph DB...$(RESET)"
	./$(BIN_PATH) status --create-indexes

.PHONY: run
run: build ## Executa o servidor de API REST localmente na porta 8080
	@echo -e "$(CYAN)⚡ Iniciando servidor de API na porta 8080...$(RESET)"
	./$(BIN_PATH) server --port 8080

##@ Dados & Ingestão
.PHONY: download-tse
download-tse: ## Baixa os dados da eleição geral de 2022 do TSE
	@echo -e "$(CYAN)📥 Baixando dados eleitorais do TSE (2022)...$(RESET)"
	./scripts/download_tse.sh 2022

.PHONY: ingest-tse
ingest-tse: build ## Inicia o pipeline de ingestão do TSE para o ano 2022
	@echo -e "$(CYAN)📦 Ingerindo candidaturas, receitas e despesas no grafo...$(RESET)"
	./$(BIN_PATH) ingest tse --year 2022 --source ./downloads/tse/2022/

##@ Consultas & Investigação
.PHONY: query-qpq
query-qpq: build ## Executa consulta de Quid Pro Quo (Doação -> Contrato pós-eleição)
	./$(BIN_PATH) query quid-pro-quo --year 2022 --min-amount 50000

.PHONY: query-ghost
query-ghost: build ## Detecta empresas recém-criadas com despesas de campanha vultosas
	./$(BIN_PATH) query ghost-suppliers --year 2022 --min-expense 100000
