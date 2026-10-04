#!/usr/bin/env bash
# ==============================================================================
# Follow-The-Fucking-Money (FTFM) - Shell Runner
# ==============================================================================
# Script de conveniência que espelha os comandos do Makefile.
# Funciona diretamente via bash mesmo sem o utilitário 'make' instalado.
# ==============================================================================

set -euo pipefail

COMPOSE_FILE="deployments/docker-compose.yml"
PROJECT_NAME="ftfm"
COMPOSE_CMD="docker compose -p ${PROJECT_NAME} -f ${COMPOSE_FILE}"
BIN_PATH="bin/ftfm"

# Cores
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'

cmd_help() {
    echo -e "${GREEN}==============================================================================${RESET}"
    echo -e "${GREEN}  Follow-The-Fucking-Money (FTFM) - Runner CLI${RESET}"
    echo -e "${GREEN}==============================================================================${RESET}"
    echo -e "Uso: ./run.sh ${CYAN}<comando>${RESET}\n"
    echo -e "${YELLOW}Infraestrutura (Docker):${RESET}"
    echo -e "  ${CYAN}up${RESET}            Sobe toda a stack (Memgraph DB + Lab UI + API Go)"
    echo -e "  ${CYAN}up-graph${RESET}      Sobe apenas Memgraph e Memgraph Lab (ideal para rodar Go local)"
    echo -e "  ${CYAN}down${RESET}          Para todos os containers"
    echo -e "  ${CYAN}down-v${RESET}        Para todos os containers e REMOVE os volumes de dados"
    echo -e "  ${CYAN}logs${RESET}          Exibe logs dos containers"
    echo -e "  ${CYAN}ps${RESET}            Lista status dos containers\n"
    echo -e "${YELLOW}Desenvolvimento & Operações:${RESET}"
    echo -e "  ${CYAN}start-all${RESET}     Sobe todo o serviço completo (Docker, DB, índices, download, ingestão e API :8075)"
    echo -e "  ${CYAN}build${RESET}         Compila o binário CLI em bin/ftfm"
    echo -e "  ${CYAN}test${RESET}          Executa os testes unitários"
    echo -e "  ${CYAN}run${RESET}           Executa o servidor de API localmente (:8075)"
    echo -e "  ${CYAN}status${RESET}        Verifica conectividade com o Graph DB e nós"
    echo -e "  ${CYAN}init-db${RESET}       Aplica constraints UNIQUE e índices no grafo\n"
    echo -e "${YELLOW}Dados & Ingestão:${RESET}"
    echo -e "  ${CYAN}download-tse${RESET}  Baixa dados eleitorais de 2022 do TSE"
    echo -e "  ${CYAN}ingest-tse${RESET}    Ingere dados de 2022 no banco de grafos"
}

cmd_build() {
    echo -e "${CYAN}🔨 Compilando o binário ${BIN_PATH}...${RESET}"
    mkdir -p bin
    go build -ldflags="-w -s" -o "${BIN_PATH}" ./cmd/ftfm
    echo -e "${GREEN}✓ Binário compilado com sucesso: ${BIN_PATH}${RESET}"
}

cmd_test() {
    echo -e "${CYAN}🧪 Executando testes unitários...${RESET}"
    go test -v -cover ./...
}

cmd_up() {
    echo -e "${CYAN}🚀 Iniciando stack FTFM...${RESET}"
    ${COMPOSE_CMD} up -d --build
    echo -e "${GREEN}✓ Stack em execução!${RESET}"
    echo -e "  • Memgraph (Bolt): ${CYAN}bolt://localhost:7687${RESET}"
    echo -e "  • Memgraph Lab (UI): ${CYAN}http://localhost:3000${RESET}"
    echo -e "  • FTFM REST API: ${CYAN}http://localhost:8080${RESET}"
}

cmd_up_graph() {
    echo -e "${CYAN}🚀 Iniciando apenas Memgraph DB e Memgraph Lab...${RESET}"
    ${COMPOSE_CMD} up -d graphdb graph-ui
    echo -e "${GREEN}✓ Banco e interface gráfica disponíveis!${RESET}"
    echo -e "  • Memgraph (Bolt): ${CYAN}bolt://localhost:7687${RESET}"
    echo -e "  • Memgraph Lab (UI): ${CYAN}http://localhost:3000${RESET}"
}

cmd_down() {
    echo -e "${YELLOW}🛑 Parando containers...${RESET}"
    ${COMPOSE_CMD} down
    echo -e "${GREEN}✓ Containers parados.${RESET}"
}

cmd_down_v() {
    echo -e "${RED}⚠️ Parando containers e apagando volumes...${RESET}"
    ${COMPOSE_CMD} down -v
    echo -e "${GREEN}✓ Volumes apagados.${RESET}"
}

cmd_status() {
    cmd_build
    ./${BIN_PATH} status
}

cmd_init_db() {
    cmd_build
    ./${BIN_PATH} status --create-indexes
}

cmd_run() {
    cmd_build
    echo -e "${CYAN}⚡ Iniciando servidor API na porta 8075...${RESET}"
    ./${BIN_PATH} server --port 8075
}

TARGET="${1:-help}"

case "${TARGET}" in
    start-all|up-all)
        ./setup_and_run.sh
        ;;
    help)
        cmd_help
        ;;
    build)
        cmd_build
        ;;
    test)
        cmd_test
        ;;
    up)
        cmd_up
        ;;
    up-graph)
        cmd_up_graph
        ;;
    down)
        cmd_down
        ;;
    down-v)
        cmd_down_v
        ;;
    logs)
        ${COMPOSE_CMD} logs -f
        ;;
    ps)
        ${COMPOSE_CMD} ps
        ;;
    status)
        cmd_status
        ;;
    init-db)
        cmd_init_db
        ;;
    run)
        cmd_run
        ;;
    download-tse)
        ./scripts/download_tse.sh 2022
        ;;
    ingest-tse)
        cmd_build
        ./${BIN_PATH} ingest tse --year 2022 --source ./downloads/tse/2022/
        ;;
    *)
        echo -e "${RED}Comando desconhecido: ${TARGET}${RESET}"
        cmd_help
        exit 1
        ;;
esac
