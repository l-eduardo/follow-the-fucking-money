#!/usr/bin/env bash
# ==============================================================================
# Follow-The-Fucking-Money (FTFM) - Setup & Run All-in-One
# ==============================================================================
# Este script executa o fluxo completo do projeto:
# 1. Compila o binário CLI em Go (bin/ftfm)
# 2. Sobe o banco Memgraph (7687) e a interface visual Memgraph Lab (3000)
# 3. Aguarda a prontidão do banco e aplica constraints e índices no grafo
# 4. Baixa os dados da eleição de 2022 do TSE (caso ainda não estejam baixados)
# 5. Ingere os dados eleitorais no grafo via streaming concorrente
# 6. Inicia a API REST exposta na porta 8075
# ==============================================================================

set -euo pipefail

# Cores para o terminal
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
BOLD='\033[1m'
RESET='\033[0m'

PORT=8075
COMPOSE_FILE="deployments/docker-compose.yml"
PROJECT_NAME="ftfm"
COMPOSE_CMD="docker compose -p ${PROJECT_NAME} -f ${COMPOSE_FILE}"
BIN_DIR="bin"
BIN_NAME="ftfm"
BIN_PATH="${BIN_DIR}/${BIN_NAME}"
DOWNLOAD_DIR="./downloads/tse/2022"

echo -e "\n${BOLD}${GREEN}==============================================================================${RESET}"
echo -e "${BOLD}${GREEN}  🚀 Follow-The-Fucking-Money (FTFM) - Inicialização Completa${RESET}"
echo -e "${BOLD}${GREEN}==============================================================================${RESET}\n"

# ------------------------------------------------------------------------------
# 1. Compilar o binário em Go
# ------------------------------------------------------------------------------
echo -e "${CYAN}▶ Passo 1/5: Compilando a CLI em Go (${BIN_PATH})...${RESET}"
mkdir -p "${BIN_DIR}"
go build -ldflags="-w -s" -o "${BIN_PATH}" ./cmd/ftfm
echo -e "${GREEN}✓ Binário compilado com sucesso!${RESET}\n"

# ------------------------------------------------------------------------------
# 2. Subir containers da infraestrutura (Memgraph DB + Memgraph Lab UI)
# ------------------------------------------------------------------------------
echo -e "${CYAN}▶ Passo 2/5: Subindo Memgraph e Memgraph Lab (Docker)...${RESET}"
${COMPOSE_CMD} up -d graphdb graph-ui

echo -e "${CYAN}⏳ Aguardando Memgraph estar pronto para conexões Bolt na porta 7687...${RESET}"
for i in {1..30}; do
    if python3 -c 'import socket; s = socket.socket(); s.connect(("127.0.0.1", 7687)); s.close()' 2>/dev/null; then
        echo -e "${GREEN}✓ Conexão Bolt ativa na porta 7687!${RESET}"
        break
    fi
    echo -n "."
    sleep 1
    if [ "$i" -eq 30 ]; then
        echo -e "\n${RED}⚠️ Timeout aguardando o Memgraph. Verifique com 'docker logs ftfm-graphdb'.${RESET}"
        exit 1
    fi
done
echo ""

# ------------------------------------------------------------------------------
# 3. Inicializar constraints e índices no banco de grafos
# ------------------------------------------------------------------------------
echo -e "${CYAN}▶ Passo 3/5: Aplicando constraints UNIQUE e índices no grafo...${RESET}"
./"${BIN_PATH}" status --create-indexes
echo -e "${GREEN}✓ Constraints e índices aplicados!${RESET}\n"

# ------------------------------------------------------------------------------
# 4. Download dos dados do TSE (se ainda não existirem)
# ------------------------------------------------------------------------------
echo -e "${CYAN}▶ Passo 4/5: Verificando dados da eleição 2022 do TSE...${RESET}"
if [ -f "${DOWNLOAD_DIR}/consulta_cand_2022.zip" ] && [ -f "${DOWNLOAD_DIR}/receitas_despesas_2022.zip" ]; then
    echo -e "${GREEN}✓ Dados do TSE já presentes em ${DOWNLOAD_DIR}!${RESET}"
else
    echo -e "${YELLOW}📥 Baixando dados públicos oficiais do TSE (2022)...${RESET}"
    ./scripts/download_tse.sh 2022
fi
echo ""

# ------------------------------------------------------------------------------
# 5. Ingestão dos dados no Grafo
# ------------------------------------------------------------------------------
echo -e "${CYAN}▶ Passo 5/5: Executando pipeline de ingestão no banco de grafos...${RESET}"
if [ -d "${DOWNLOAD_DIR}" ]; then
    ./"${BIN_PATH}" ingest tse --year 2022 --source "${DOWNLOAD_DIR}" --batch-size 5000 --workers 8 || echo -e "${YELLOW}⚠️ Aviso na ingestão (dados parciais carregados)${RESET}"
fi

echo -e "\n${BOLD}${GREEN}==============================================================================${RESET}"
echo -e "${BOLD}${GREEN}  🎉 SERVIÇOS FTFM PRONTOS E OPERACIONAIS!${RESET}"
echo -e "${BOLD}${GREEN}==============================================================================${RESET}"
echo -e "  🌐 ${BOLD}REST API (Porta ${PORT}):${RESET}        ${CYAN}http://localhost:${PORT}${RESET}"
echo -e "  📊 ${BOLD}Memgraph Lab UI (Porta 3000):${RESET}  ${CYAN}http://localhost:3000${RESET}"
echo -e "  🔗 ${BOLD}Protocolo Bolt:${RESET}                 ${CYAN}bolt://localhost:7687${RESET}"
echo -e "\n${YELLOW}${BOLD}Exemplos de testes da API via terminal:${RESET}"
echo -e "  curl http://localhost:${PORT}/health"
echo -e "  curl http://localhost:${PORT}/api/v1/stats"
echo -e "  curl \"http://localhost:${PORT}/api/v1/trail?from=00000000000191&to=LULA\""
echo -e "  curl \"http://localhost:${PORT}/api/v1/patterns/quid-pro-quo?year=2022\""
echo -e "  curl \"http://localhost:${PORT}/api/v1/patterns/ghost-suppliers?year=2022\""
echo -e "${BOLD}${GREEN}==============================================================================${RESET}\n"

echo -e "${CYAN}⚡ Iniciando servidor de API na porta ${PORT} (Ctrl+C para encerrar)...${RESET}\n"
./"${BIN_PATH}" server --port "${PORT}"
