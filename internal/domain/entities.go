package domain

import (
	"time"
)

// DataLineage provides strict auditability for legal compliance and data provenance.
type DataLineage struct {
	SourceSystem  string    `json:"source_system"`
	SourceURL     string    `json:"source_url"`
	FileChecksum  string    `json:"file_checksum,omitempty"`
	RawLineNumber int64     `json:"raw_line_number,omitempty"`
	ExtractedAt   time.Time `json:"extracted_at"`
}

// Person represents an individual identified in TSE, RFB QSA, or public records.
type Person struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	MaskedCPF  string      `json:"masked_cpf"`
	Middle6CPF string      `json:"middle6_cpf"`
	State      string      `json:"state,omitempty"`
	AgeBracket string      `json:"age_bracket,omitempty"`
	Flags      []string    `json:"flags,omitempty"` // e.g. ["DOADOR", "SOCIO", "CANDIDATO"]
	Lineage    DataLineage `json:"lineage"`
}

// Company represents a legal entity (PJ) from RFB, TSE, or PNCP.
type Company struct {
	CNPJ             string      `json:"cnpj"`
	CNPJRoot         string      `json:"cnpj_root"`
	LegalName        string      `json:"legal_name"`
	TradeName        string      `json:"trade_name,omitempty"`
	ShareCapital     float64     `json:"share_capital"`
	PrimaryCNAE      string      `json:"primary_cna"`
	CadastralStatus  string      `json:"cadastral_status"` // e.g. "ATIVA", "BAIXADA"
	OpeningDate      string      `json:"opening_date,omitempty"`
	State            string      `json:"state"`
	City             string      `json:"city"`
	IsHeadquarter    bool        `json:"is_headquarter"`
	Lineage          DataLineage `json:"lineage"`
}

// Candidate represents an electoral candidate in TSE.
type Candidate struct {
	SQCandidate     string      `json:"sq_candidate"` // Primary electoral sequential ID
	BallotName      string      `json:"ballot_name"`
	FullName        string      `json:"full_name"`
	Number          int         `json:"number"`
	Office          string      `json:"office"`       // e.g. "PRESIDENTE", "GOVERNADOR"
	ElectionYear    int         `json:"election_year"`
	Party           string      `json:"party"`
	State           string      `json:"state"`
	City            string      `json:"city,omitempty"`
	ResultStatus    string      `json:"result_status"` // e.g. "ELEITO", "NAO ELEITO"
	TotalAssets     float64     `json:"total_assets,omitempty"`
	Lineage         DataLineage `json:"lineage"`
}

// CandidateAsset represents an asset (bem) declared by a candidate to the TSE.
type CandidateAsset struct {
	ID           string      `json:"id"`
	SQCandidate  string      `json:"sq_candidate"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	Value        float64     `json:"value"`
	ElectionYear int         `json:"election_year"`
	Lineage      DataLineage `json:"lineage"`
}

// Donation represents a campaign contribution to a candidate or party.
type Donation struct {
	IDTransaction  string      `json:"id_transaction"`
	DonorType      string      `json:"donor_type"` // "PF" or "PJ"
	DonorID        string      `json:"donor_id"`
	DonorName      string      `json:"donor_name"`
	RecipientSQ    string      `json:"recipient_sq"`
	RecipientName  string      `json:"recipient_name"`
	Amount         float64     `json:"amount"`
	DonationDate   string      `json:"donation_date"`
	ReceiptNumber  string      `json:"receipt_number,omitempty"`
	RevenueType    string      `json:"revenue_type,omitempty"`
	ElectionYear   int         `json:"election_year"`
	Lineage        DataLineage `json:"lineage"`
}

// Expense represents a campaign expenditure paid to a supplier.
type Expense struct {
	IDTransaction  string      `json:"id_transaction"`
	CandidateSQ    string      `json:"candidate_sq"`
	SupplierCNPJ   string      `json:"supplier_cnpj"`
	SupplierName   string      `json:"supplier_name"`
	Amount         float64     `json:"amount"`
	ExpenseDate    string      `json:"expense_date"`
	Description    string      `json:"description,omitempty"`
	InvoiceNumber  string      `json:"invoice_number,omitempty"`
	ElectionYear   int         `json:"election_year"`
	Lineage        DataLineage `json:"lineage"`
}

// PublicContract represents a contract signed between government agencies and companies.
type PublicContract struct {
	IDContract       string      `json:"id_contract"`
	ContractNumber   string      `json:"contract_number"`
	Year             int         `json:"year"`
	AgencyUG         string      `json:"agency_ug"`
	AgencyName       string      `json:"agency_name"`
	ContractorCNPJ   string      `json:"contractor_cnpj"`
	ContractorName   string      `json:"contractor_name"`
	InitialAmount    float64     `json:"initial_amount"`
	FinalAmount      float64     `json:"final_amount"`
	SignatureDate    string      `json:"signature_date"`
	EndDate          string      `json:"end_date,omitempty"`
	ObjectSummary    string      `json:"object_summary"`
	Lineage          DataLineage `json:"lineage"`
}

// TrailHop represents a single step in a financial or relational trail graph.
type TrailHop struct {
	FromLabel    string         `json:"from_label"`
	FromID       string         `json:"from_id"`
	FromName     string         `json:"from_name"`
	Relationship string         `json:"relationship"`
	Properties   map[string]any `json:"properties"`
	ToLabel      string         `json:"to_label"`
	ToID         string         `json:"to_id"`
	ToName       string         `json:"to_name"`
}

// TrailResult represents the shortest or all paths between two entities.
type TrailResult struct {
	FromQuery   string     `json:"from_query"`
	ToQuery     string     `json:"to_query"`
	HopsCount   int        `json:"hops_count"`
	TotalAmount float64    `json:"total_amount,omitempty"`
	Path        []TrailHop `json:"path"`
}
