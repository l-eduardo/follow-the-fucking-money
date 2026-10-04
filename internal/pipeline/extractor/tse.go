package extractor

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/l-eduardo/follow-the-fucking-money/internal/domain"
	"github.com/l-eduardo/follow-the-fucking-money/pkg/brazil"
	"github.com/l-eduardo/follow-the-fucking-money/pkg/streamutil"
)

// ParseTSECandidate parses a row from consulta_cand_{YEAR}_{UF}.csv
// Expected column indices vary slightly by year, but core columns are matched.
func ParseTSECandidate(record []string, year int) (*domain.Candidate, error) {
	if len(record) < 30 {
		return nil, fmt.Errorf("insufficient columns in candidate record: %d", len(record))
	}

	// Typical TSE layout:
	// Col 15: SQ_CANDIDATO, Col 16: NR_CANDIDATO, Col 17: NM_CANDIDATO, Col 18: NM_URNA_CANDIDATO
	// Col 14: DS_CARGO, Col 27: SG_PARTIDO, Col 11: SG_UF, Col 56: DS_SIT_TOT_TURNO
	sq := strings.TrimSpace(record[15])
	if sq == "" || sq == "-1" {
		return nil, fmt.Errorf("invalid SQ_CANDIDATO")
	}

	number, _ := strconv.Atoi(strings.TrimSpace(record[16]))

	return &domain.Candidate{
		SQCandidate:  sq,
		BallotName:   streamutil.NormalizeText(record[18]),
		FullName:     streamutil.NormalizeText(record[17]),
		Number:       number,
		Office:       streamutil.NormalizeText(record[14]),
		ElectionYear: year,
		Party:        strings.ToUpper(strings.TrimSpace(record[27])),
		State:        strings.ToUpper(strings.TrimSpace(record[11])),
		ResultStatus: streamutil.NormalizeText(record[len(record)-1]),
		Lineage: domain.DataLineage{
			SourceSystem: "TSE_CONSULTA_CAND",
			ExtractedAt:  time.Now(),
		},
	}, nil
}

// ParseTSEDonation parses a row from receitas_candidatos_{YEAR}_{UF}.csv
func ParseTSEDonation(record []string, year int) (*domain.Donation, error) {
	if len(record) < 35 {
		return nil, fmt.Errorf("insufficient columns in donation record: %d", len(record))
	}

	// Col 16: SQ_CANDIDATO, Col 17: NM_CANDIDATO
	// Col 24: NR_CPF_CNPJ_DOADOR, Col 25: NM_DOADOR
	// Col 30: VR_RECEITA, Col 28: DT_RECEITA
	candidateSQ := strings.TrimSpace(record[16])
	donorDoc := brazil.CleanCPF(record[24])
	if candidateSQ == "" || donorDoc == "" {
		return nil, fmt.Errorf("empty candidate or donor ID")
	}

	donorType := "PF"
	if len(donorDoc) == 14 {
		donorType = "PJ"
	}

	valStr := strings.ReplaceAll(strings.TrimSpace(record[30]), ",", ".")
	amount, _ := strconv.ParseFloat(valStr, 64)

	txID := fmt.Sprintf("DON_%d_%s_%s_%d", year, candidateSQ, donorDoc, int64(amount*100))

	return &domain.Donation{
		IDTransaction: txID,
		DonorType:     donorType,
		DonorID:       donorDoc,
		DonorName:     streamutil.NormalizeText(record[25]),
		RecipientSQ:   candidateSQ,
		RecipientName: streamutil.NormalizeText(record[17]),
		Amount:        amount,
		DonationDate:  strings.TrimSpace(record[28]),
		ElectionYear:  year,
		Lineage: domain.DataLineage{
			SourceSystem: "TSE_RECEITAS_CANDIDATOS",
			ExtractedAt:  time.Now(),
		},
	}, nil
}

// ParseTSEExpense parses a row from despesas_contratadas_candidatos_{YEAR}_{UF}.csv
func ParseTSEExpense(record []string, year int) (*domain.Expense, error) {
	if len(record) < 30 {
		return nil, fmt.Errorf("insufficient columns in expense record: %d", len(record))
	}

	candidateSQ := strings.TrimSpace(record[16])
	supplierCNPJ := brazil.CleanCNPJ(record[25])
	if candidateSQ == "" || supplierCNPJ == "" {
		return nil, fmt.Errorf("empty candidate or supplier ID")
	}

	valStr := strings.ReplaceAll(strings.TrimSpace(record[29]), ",", ".")
	amount, _ := strconv.ParseFloat(valStr, 64)

	txID := fmt.Sprintf("EXP_%d_%s_%s_%d", year, candidateSQ, supplierCNPJ, int64(amount*100))

	return &domain.Expense{
		IDTransaction: txID,
		CandidateSQ:   candidateSQ,
		SupplierCNPJ:  supplierCNPJ,
		SupplierName:  streamutil.NormalizeText(record[26]),
		Amount:        amount,
		ExpenseDate:   strings.TrimSpace(record[28]),
		Description:   streamutil.NormalizeText(record[len(record)-3]),
		ElectionYear:  year,
		Lineage: domain.DataLineage{
			SourceSystem: "TSE_DESPESAS_CANDIDATOS",
			ExtractedAt:  time.Now(),
		},
	}, nil
}

// ParseTSEAsset parses a row from bem_candidato_{YEAR}_{UF}.csv
func ParseTSEAsset(record []string, year int) (*domain.CandidateAsset, error) {
	if len(record) < 17 {
		return nil, fmt.Errorf("insufficient columns in asset record: %d", len(record))
	}

	// Col 11: SQ_CANDIDATO, Col 14: DS_TIPO_BEM_CANDIDATO, Col 15: DS_BEM_CANDIDATO, Col 16: VR_BEM_CANDIDATO
	sq := strings.TrimSpace(record[11])
	if sq == "" || sq == "-1" {
		return nil, fmt.Errorf("invalid SQ_CANDIDATO")
	}

	valStr := strings.ReplaceAll(strings.TrimSpace(record[16]), ",", ".")
	val, _ := strconv.ParseFloat(valStr, 64)

	order := strings.TrimSpace(record[12])
	assetID := fmt.Sprintf("BEM_%s_%s", sq, order)

	return &domain.CandidateAsset{
		ID:           assetID,
		SQCandidate:  sq,
		Type:         streamutil.NormalizeText(record[14]),
		Description:  streamutil.NormalizeText(record[15]),
		Value:        val,
		ElectionYear: year,
		Lineage: domain.DataLineage{
			SourceSystem: "TSE_BEM_CANDIDATO",
			ExtractedAt:  time.Now(),
		},
	}, nil
}

