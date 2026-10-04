#!/usr/bin/env bash
# ==============================================================================
# Follow-The-Fucking-Money (FTFM) - Clean All / Purge
# ==============================================================================
# Limpa completamente o ambiente do projeto:
# 1. Para e remove todos os containers Docker
# 2. Remove todos os volumes (banco de dados, logs)
# 3. Remove todas as imagens Docker do projeto (Memgraph, Lab, FTFM API)
# 4. Apaga todos os arquivos baixados do TSE/RFB em downloads/
# 5. Preserva a estrutura de pastas downloads/ e downloads/tse/ via .gitkeep
# 6. Remove binários compilados em bin/ e relatórios de cobertura
# ==============================================================================

set -euo pipefail

CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
BOLD='\033[1m'
RESET='\033[0m'

COMPOSE_FILE="deployments/docker-compose.yml"
PROJECT_NAME="ftfm"
COMPOSE_CMD="docker compose -p ${PROJECT_NAME} -f ${COMPOSE_FILE}"

echo -e "\n${BOLD}${RED}==============================================================================${RESET}"
echo -e "${BOLD}${RED}  🧹 FTFM - Limpeza Completa (Purge All)${RESET}"
echo -e "${BOLD}${RED}==============================================================================${RESET}\n"

# 1. Docker: Containers, Volumes e Imagens
echo -e "${YELLOW}🛑 Passo 1/3: Parando e removendo containers, volumes e imagens Docker...${RESET}"
${COMPOSE_CMD} down -v --rmi all --remove-orphans 2>/dev/null || true

# Remoção forçada de containers ou volumes residuais do projeto
docker rm -f ftfm-graphdb ftfm-graph-ui ftfm-api 2>/dev/null || true
docker volume rm -f ftfm_mg_data ftfm_mg_log deployments_mg_data deployments_mg_log 2>/dev/null || true
echo -e "${GREEN}✓ Containers, volumes e imagens removidos com sucesso!${RESET}\n"

# 2. Dados baixados (downloads/)
echo -e "${YELLOW}🗑️ Passo 2/3: Removendo arquivos baixados em downloads/...${RESET}"
if [ -d "downloads" ]; then
    find downloads/ -type f ! -name '.gitkeep' -delete 2>/dev/null || true
    find downloads/ -depth -type d -not -path 'downloads' -not -path 'downloads/tse' -empty -delete 2>/dev/null || true
fi

# Garante que a estrutura de diretórios do repositório continue no git
mkdir -p downloads/tse
touch downloads/.gitkeep
touch downloads/tse/.gitkeep
echo -e "${GREEN}✓ Arquivos baixados removidos! Pastas mantidas no git com .gitkeep.${RESET}\n"

# 3. Binários e caches locais
echo -e "${YELLOW}🧹 Passo 3/3: Removendo binários e relatórios temporários...${RESET}"
rm -rf bin/ dist/ ftfm cover.out coverage.html data/
echo -e "${GREEN}✓ Binários e caches limpos!${RESET}\n"

echo -e "${BOLD}${GREEN}==============================================================================${RESET}"
echo -e "${BOLD}${GREEN}  ✓ AMBIENTE 100% LIMPO E RESETADO!${RESET}"
echo -e "${BOLD}${GREEN}==============================================================================${RESET}\n"
