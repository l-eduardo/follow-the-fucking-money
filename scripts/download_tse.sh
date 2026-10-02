#!/usr/bin/env bash
set -euo pipefail

YEAR="${1:-2022}"
DEST_DIR="${2:-./downloads/tse/${YEAR}}"

echo "=================================================="
echo "📥 FTFM - Download de Dados Eleitorais do TSE (${YEAR})"
echo "📁 Diretório de destino: ${DEST_DIR}"
echo "=================================================="

mkdir -p "${DEST_DIR}"

BASE_URL="https://cdn.tse.jus.br/estatistica/sead/odsele"

download_file() {
    local url="$1"
    local filename="$2"
    local target="${DEST_DIR}/${filename}"

    if [ -f "${target}" ]; then
        echo "⏭️ Arquivo ${filename} já existe, pulando download."
    else
        echo "⬇️ Baixando ${filename}..."
        curl -fSL -o "${target}" "${url}" || echo "⚠️ Aviso: Falha ao baixar ${filename} de ${url}"
    fi
}

# 1. Candidaturas
download_file "${BASE_URL}/consulta_cand/consulta_cand_${YEAR}.zip" "consulta_cand_${YEAR}.zip"

# 2. Declaração de Bens
download_file "${BASE_URL}/bem_candidato/bem_candidato_${YEAR}.zip" "bem_candidato_${YEAR}.zip"

# 3. Prestação de Contas - Receitas e Despesas
download_file "${BASE_URL}/prestacao_contas/prestacao_de_contas_eleitorais_candidatos_${YEAR}.zip" "receitas_despesas_${YEAR}.zip"

echo "✅ Downloads concluídos para o ano ${YEAR} em ${DEST_DIR}"
echo "💡 Para ingerir no grafo execute:"
echo "   ./bin/ftfm ingest tse --year ${YEAR} --source ${DEST_DIR}"
