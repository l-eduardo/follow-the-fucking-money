#!/usr/bin/env bash
# ==============================================================================
# Follow-The-Fucking-Money (FTFM) - Download TSE Dataset
# ==============================================================================
set -euo pipefail

YEAR="${1:-2026}"
DEST_DIR="${2:-./downloads/tse/${YEAR}}"
QUICK_MODE="${QUICK:-0}"

# Process flags
for arg in "$@"; do
    if [ "$arg" = "--quick" ]; then
        QUICK_MODE=1
    fi
done

CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RESET='\033[0m'

echo -e "\n=================================================="
echo -e "📥 FTFM - Download de Dados Eleitorais do TSE (${YEAR})"
echo -e "📁 Diretório de destino: ${DEST_DIR}"
echo -e "==================================================\n"

mkdir -p "${DEST_DIR}"

BASE_URL="https://cdn.tse.jus.br/estatistica/sead/odsele"

is_valid_zip() {
    local file="$1"
    if [ ! -f "${file}" ]; then
        return 1
    fi
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import zipfile, sys; sys.exit(0 if zipfile.is_zipfile(sys.argv[1]) else 1)" "${file}" 2>/dev/null
    else
        unzip -t -q "${file}" 2>/dev/null
    fi
}

download_file() {
    local url="$1"
    local filename="$2"
    local desc="$3"
    local target="${DEST_DIR}/${filename}"

    if is_valid_zip "${target}"; then
        echo -e "${GREEN}⏭️ Arquivo ${filename} já existe e é válido, pulando download.${RESET}"
    else
        echo -e "${CYAN}⬇️ Baixando ${filename} (${desc})...${RESET}"
        if [ -f "${target}" ]; then
            echo -e "${YELLOW}↪️ Arquivo parcial detectado. Retomando download de onde parou...${RESET}"
        fi

        echo -e "${YELLOW}📊 Acompanhamento de progresso em tempo real (%, velocidade, tempo estimado):${RESET}"
        if command -v wget >/dev/null 2>&1; then
            wget -c -q --show-progress --tries=3 -O "${target}" "${url}" || {
                echo -e "${YELLOW}⚠️ Aviso: Falha ao baixar ${filename} com wget. Tentando com curl...${RESET}"
                curl -fL --progress-bar -C - --retry 3 --retry-delay 2 -o "${target}" "${url}" || {
                    echo -e "${YELLOW}⚠️ Erro ao baixar ${filename} de ${url}${RESET}"
                    return 0
                }
            }
        else
            curl -fL --progress-bar -C - --retry 3 --retry-delay 2 -o "${target}" "${url}" || {
                echo -e "${YELLOW}⚠️ Erro ao baixar ${filename} de ${url}${RESET}"
                return 0
            }
        fi

        if is_valid_zip "${target}"; then
            echo -e "${GREEN}✓ Download concluído e verificado com sucesso: ${filename}${RESET}"
        else
            echo -e "${YELLOW}⚠️ Download concluído, mas o arquivo zip parece parcial ou corrompido.${RESET}"
        fi
    fi
}

# 1. Candidaturas (~4.3 MB)
download_file "${BASE_URL}/consulta_cand/consulta_cand_${YEAR}.zip" \
    "consulta_cand_${YEAR}.zip" \
    "~4.3 MB - Candidatos e Partidos"

# 2. Declaração de Bens (~5.3 MB)
download_file "${BASE_URL}/bem_candidato/bem_candidato_${YEAR}.zip" \
    "bem_candidato_${YEAR}.zip" \
    "~5.3 MB - Bens Declarados"

# 3. Prestação de Contas - Receitas e Despesas (~440 MB)
if [ "${QUICK_MODE}" = "1" ]; then
    echo -e "${YELLOW}⏩ Modo rápido ativado (--quick): pulando arquivo pesado de receitas e despesas (~440 MB).${RESET}"
else
    echo -e "${CYAN}ℹ️ O próximo arquivo contém todas as receitas e despesas eleitorais de ${YEAR} (~440 MB).${RESET}"
    echo -e "${CYAN}   O download pode levar alguns minutos dependendo da sua conexão.${RESET}"
    download_file "${BASE_URL}/prestacao_contas/prestacao_de_contas_eleitorais_candidatos_${YEAR}.zip" \
        "receitas_despesas_${YEAR}.zip" \
        "~440 MB - Doações e Despesas"
fi

echo -e "\n${GREEN}✅ Downloads concluídos para o ano ${YEAR} em ${DEST_DIR}${RESET}"
echo -e "💡 Para ingerir no grafo execute:"
echo -e "   ./bin/ftfm ingest tse --year ${YEAR} --source ${DEST_DIR}\n"
