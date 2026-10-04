#!/usr/bin/env bash
# ==============================================================================
# Follow-The-Fucking-Money (FTFM) - Setup & Run All-in-One
# ==============================================================================
# Este script executa o fluxo completo do projeto:
# 1. Compila o binário CLI em Go (bin/ftfm)
# 2. Sobe o banco Memgraph (7687) e a interface visual Memgraph Lab (3000)
# 3. Aguarda a prontidão real do banco e aplica constraints e índices no grafo
# 4. Baixa os dados da eleição de 2022 do TSE (com barra de progresso e resume)
# 5. Ingere os dados eleitorais no grafo via streaming concorrente
# 6. Inicia a API REST exposta na porta 8075
# ==============================================================================

set -euo pipefail
trap 'echo -e "\n\033[31m❌ Ocorreu uma interrupção ou erro na linha $LINENO.\033[0m"' ERR

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

YEAR="2026"
if [[ "${1:-}" =~ ^[0-9]{4}$ ]]; then
    YEAR="$1"
    shift
fi
DOWNLOAD_DIR="./downloads/tse/${YEAR}"
EXTRA_ARGS=("$@")

echo -e "\n${BOLD}${GREEN}==============================================================================${RESET}"
echo -e "${BOLD}${GREEN}  🚀 Follow-The-Fucking-Money (FTFM) - Inicialização Completa (Eleição ${YEAR})${RESET}"
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

echo -e "${CYAN}⏳ Aguardando Memgraph inicializar e responder ao protocolo Bolt (7687)...${RESET}"
READY=false
for i in $(seq 1 45); do
    if ./"${BIN_PATH}" status >/dev/null 2>&1; then
        READY=true
        echo -e "\n${GREEN}✓ Memgraph online e respondendo na porta 7687!${RESET}"
        break
    fi
    echo -n "."
    sleep 1
done

if [ "${READY}" != "true" ]; then
    echo -e "\n${RED}⚠️ Timeout aguardando o Memgraph. Verifique com 'docker logs ftfm-graphdb'.${RESET}"
    exit 1
fi
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
echo -e "${CYAN}▶ Passo 4/5: Verificando dados da eleição ${YEAR} do TSE...${RESET}"
./scripts/download_tse.sh "${YEAR}" "${DOWNLOAD_DIR}" "${EXTRA_ARGS[@]}"
echo ""

# ------------------------------------------------------------------------------
# 5. Ingestão dos dados no Grafo
# ------------------------------------------------------------------------------
echo -e "${CYAN}▶ Passo 5/5: Executando pipeline de ingestão no banco de grafos...${RESET}"
if [ -d "${DOWNLOAD_DIR}" ] && [ "$(find "${DOWNLOAD_DIR}" -maxdepth 1 -name '*.zip' -o -name '*.csv' | wc -l)" -gt 0 ]; then
    ./"${BIN_PATH}" ingest tse --year "${YEAR}" --source "${DOWNLOAD_DIR}" --batch-size 5000 --workers 8 || echo -e "${YELLOW}⚠️ Aviso na ingestão (dados parciais carregados)${RESET}"
else
    echo -e "${YELLOW}ℹ️ Nenhum arquivo encontrado em ${DOWNLOAD_DIR} para ingestão imediata.${RESET}"
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
echo -e "  curl \"http://localhost:${PORT}/api/v1/patterns/quid-pro-quo?year=${YEAR}\""
echo -e "  curl \"http://localhost:${PORT}/api/v1/patterns/ghost-suppliers?year=${YEAR}\""
echo -e "${BOLD}${GREEN}==============================================================================${RESET}\n"

echo -e "${CYAN}⚡ Iniciando servidor de API na porta ${PORT} (Ctrl+C para encerrar)...${RESET}\n"
./"${BIN_PATH}" server --port "${PORT}"
