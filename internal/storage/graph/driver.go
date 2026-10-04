package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/l-eduardo/follow-the-fucking-money/internal/domain"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

// Service implements domain.GraphRepository and domain.GraphQueryService.
type Service struct {
	driver neo4j.DriverWithContext
	dbName string
}

// NewService instantiates a Bolt connection pool for Memgraph or Neo4j.
func NewService(uri, user, pass, dbName string) (*Service, error) {
	var auth neo4j.AuthToken
	if user != "" || pass != "" {
		auth = neo4j.BasicAuth(user, pass, "")
	} else {
		auth = neo4j.NoAuth()
	}

	driver, err := neo4j.NewDriverWithContext(uri, auth, func(config *neo4j.Config) {
		config.MaxConnectionPoolSize = 50
		config.ConnectionAcquisitionTimeout = 15 * time.Second
		config.SocketKeepalive = true
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create graph driver: %w", err)
	}

	return &Service{
		driver: driver,
		dbName: dbName,
	}, nil
}

// Ping verifies connectivity to the graph engine.
func (s *Service) Ping(ctx context.Context) error {
	return s.driver.VerifyConnectivity(ctx)
}

// Close gracefully closes the driver connection pool.
func (s *Service) Close(ctx context.Context) error {
	return s.driver.Close(ctx)
}

// CreateConstraintsAndIndexes applies uniqueness constraints and lookup indexes using auto-commit transactions.
func (s *Service) CreateConstraintsAndIndexes(ctx context.Context) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	queries := []string{
		"CREATE CONSTRAINT ON (c:Candidate) ASSERT c.sq_candidate IS UNIQUE",
		"CREATE CONSTRAINT ON (p:Person) ASSERT p.id IS UNIQUE",
		"CREATE CONSTRAINT ON (e:Company) ASSERT e.cnpj IS UNIQUE",
		"CREATE CONSTRAINT ON (a:Asset) ASSERT a.id IS UNIQUE",
		"CREATE CONSTRAINT ON (cp:PublicContract) ASSERT cp.id_contract IS UNIQUE",
		"CREATE CONSTRAINT ON (o:PublicAgency) ASSERT o.ug IS UNIQUE",
		"CREATE INDEX ON :Candidate(sq_candidate)",
		"CREATE INDEX ON :Person(id)",
		"CREATE INDEX ON :Company(cnpj)",
		"CREATE INDEX ON :Asset(id)",
		"CREATE INDEX ON :PublicContract(id_contract)",
		"CREATE INDEX ON :PublicAgency(ug)",
		"CREATE INDEX ON :Company(cnpj_root)",
		"CREATE INDEX ON :Person(masked_cpf)",
		"CREATE INDEX ON :Candidate(ballot_name)",
	}

	for _, q := range queries {
		res, err := session.Run(ctx, q, nil)
		if err != nil {
			log.Debug().Err(err).Str("query", q).Msg("Constraint/Index notice (may already exist)")
			continue
		}
		if _, err := res.Consume(ctx); err != nil {
			log.Debug().Err(err).Str("query", q).Msg("Constraint/Index consume notice")
		}
	}

	log.Info().Msg("Graph constraints and indexes verified")
	return nil
}

// BatchInsertCandidates inserts or merges candidates in a single transaction using UNWIND.
func (s *Service) BatchInsertCandidates(ctx context.Context, candidates []domain.Candidate) error {
	if len(candidates) == 0 {
		return nil
	}

	rows := make([]map[string]any, len(candidates))
	for i, c := range candidates {
		rows[i] = map[string]any{
			"sq":          c.SQCandidate,
			"ballot_name": c.BallotName,
			"full_name":   c.FullName,
			"number":      c.Number,
			"office":      c.Office,
			"year":        c.ElectionYear,
			"party":       c.Party,
			"state":       c.State,
			"city":        c.City,
			"status":      c.ResultStatus,
			"assets":      c.TotalAssets,
		}
	}

	query := `
		UNWIND $batch AS row
		MERGE (c:Candidate {sq_candidate: row.sq})
		ON CREATE SET
			c.ballot_name = row.ballot_name,
			c.full_name = row.full_name,
			c.number = row.number,
			c.office = row.office,
			c.year = row.year,
			c.party = row.party,
			c.state = row.state,
			c.city = row.city,
			c.status = row.status,
			c.total_assets = row.assets,
			c.updated_at = datetime()
	`

	return s.executeWriteBatch(ctx, query, rows)
}

// BatchInsertAssets creates Asset nodes and connects them to Candidate via POSSUI_BEM.
func (s *Service) BatchInsertAssets(ctx context.Context, assets []domain.CandidateAsset) error {
	if len(assets) == 0 {
		return nil
	}

	rows := make([]map[string]any, len(assets))
	for i, a := range assets {
		rows[i] = map[string]any{
			"id":   a.ID,
			"sq":   a.SQCandidate,
			"type": a.Type,
			"desc": a.Description,
			"val":  a.Value,
			"year": a.ElectionYear,
		}
	}

	query := `
		UNWIND $batch AS row
		MATCH (c:Candidate {sq_candidate: row.sq})
		MERGE (b:Asset {id: row.id})
		ON CREATE SET
			b.type = row.type,
			b.description = row.desc,
			b.value = row.val,
			b.year = row.year
		MERGE (c)-[:POSSUI_BEM]->(b)
	`

	return s.executeWriteBatch(ctx, query, rows)
}

// BatchInsertDonations merges donors and creates DOOU_PARA relationships.
func (s *Service) BatchInsertDonations(ctx context.Context, donations []domain.Donation) error {
	if len(donations) == 0 {
		return nil
	}

	rows := make([]map[string]any, len(donations))
	for i, d := range donations {
		rows[i] = map[string]any{
			"id_tx":         d.IDTransaction,
			"donor_type":    d.DonorType,
			"donor_id":      d.DonorID,
			"donor_name":    d.DonorName,
			"recipient_sq":  d.RecipientSQ,
			"amount":        d.Amount,
			"donation_date": d.DonationDate,
			"receipt":       d.ReceiptNumber,
			"rev_type":      d.RevenueType,
			"year":          d.ElectionYear,
		}
	}

	query := `
		UNWIND $batch AS row
		MATCH (c:Candidate {sq_candidate: row.recipient_sq})
		FOREACH (_ IN CASE WHEN row.donor_type = 'PF' THEN [1] ELSE [] END |
			MERGE (p:Person {id: row.donor_id})
			ON CREATE SET p.name = row.donor_name
			MERGE (p)-[r:DOOU_PARA {id_tx: row.id_tx}]->(c)
			ON CREATE SET r.amount = row.amount, r.date = row.donation_date, r.year = row.year
		)
		FOREACH (_ IN CASE WHEN row.donor_type = 'PJ' THEN [1] ELSE [] END |
			MERGE (comp:Company {cnpj: row.donor_id})
			ON CREATE SET comp.legal_name = row.donor_name
			MERGE (comp)-[r:DOOU_PARA {id_tx: row.id_tx}]->(c)
			ON CREATE SET r.amount = row.amount, r.date = row.donation_date, r.year = row.year
		)
	`

	return s.executeWriteBatch(ctx, query, rows)
}

// BatchInsertExpenses merges suppliers and creates FORNECEU_PARA relationships.
func (s *Service) BatchInsertExpenses(ctx context.Context, expenses []domain.Expense) error {
	if len(expenses) == 0 {
		return nil
	}

	rows := make([]map[string]any, len(expenses))
	for i, e := range expenses {
		suppType := "PF"
		if len(e.SupplierCNPJ) == 14 {
			suppType = "PJ"
		}

		rows[i] = map[string]any{
			"id_tx":         e.IDTransaction,
			"candidate_sq":  e.CandidateSQ,
			"supplier_doc":  e.SupplierCNPJ,
			"supplier_name": e.SupplierName,
			"supplier_type": suppType,
			"amount":        e.Amount,
			"date":          e.ExpenseDate,
			"description":   e.Description,
			"invoice":       e.InvoiceNumber,
			"year":          e.ElectionYear,
		}
	}

	query := `
		UNWIND $batch AS row
		MATCH (c:Candidate {sq_candidate: row.candidate_sq})
		FOREACH (_ IN CASE WHEN row.supplier_type = 'PF' THEN [1] ELSE [] END |
			MERGE (p:Person {id: row.supplier_doc})
			ON CREATE SET p.name = row.supplier_name
			MERGE (p)-[r:FORNECEU_PARA {id_tx: row.id_tx}]->(c)
			ON CREATE SET
				r.amount = row.amount,
				r.date = row.date,
				r.description = row.description,
				r.invoice = row.invoice,
				r.year = row.year
		)
		FOREACH (_ IN CASE WHEN row.supplier_type = 'PJ' THEN [1] ELSE [] END |
			MERGE (comp:Company {cnpj: row.supplier_doc})
			ON CREATE SET comp.legal_name = row.supplier_name
			MERGE (comp)-[r:FORNECEU_PARA {id_tx: row.id_tx}]->(c)
			ON CREATE SET
				r.amount = row.amount,
				r.date = row.date,
				r.description = row.description,
				r.invoice = row.invoice,
				r.year = row.year
		)
	`

	return s.executeWriteBatch(ctx, query, rows)
}

// BatchInsertCompanies merges company nodes.
func (s *Service) BatchInsertCompanies(ctx context.Context, companies []domain.Company) error {
	if len(companies) == 0 {
		return nil
	}

	rows := make([]map[string]any, len(companies))
	for i, c := range companies {
		rows[i] = map[string]any{
			"cnpj":       c.CNPJ,
			"cnpj_root":  c.CNPJRoot,
			"legal_name": c.LegalName,
			"trade_name": c.TradeName,
			"capital":    c.ShareCapital,
			"cnae":       c.PrimaryCNAE,
			"status":     c.CadastralStatus,
			"opening":    c.OpeningDate,
			"state":      c.State,
			"city":       c.City,
			"is_matrix":  c.IsHeadquarter,
		}
	}

	query := `
		UNWIND $batch AS row
		MERGE (comp:Company {cnpj: row.cnpj})
		ON CREATE SET
			comp.cnpj_root = row.cnpj_root,
			comp.legal_name = row.legal_name,
			comp.trade_name = row.trade_name,
			comp.share_capital = row.capital,
			comp.primary_cnae = row.cnae,
			comp.status = row.status,
			comp.opening_date = row.opening,
			comp.state = row.state,
			comp.city = row.city,
			comp.is_matrix = row.is_matrix
	`

	return s.executeWriteBatch(ctx, query, rows)
}

// BatchInsertPartners merges corporate partners and SOCIO_DE relationships.
func (s *Service) BatchInsertPartners(ctx context.Context, partners []map[string]any) error {
	if len(partners) == 0 {
		return nil
	}

	query := `
		UNWIND $batch AS row
		MERGE (p:Person {id: row.person_id})
		ON CREATE SET p.name = row.person_name, p.masked_cpf = row.masked_cpf
		MATCH (comp:Company {cnpj: row.company_cnpj})
		MERGE (p)-[r:SOCIO_DE]->(comp)
		ON CREATE SET
			r.qualification = row.qualification,
			r.entry_date = row.entry_date
	`

	return s.executeWriteBatch(ctx, query, partners)
}

// BatchInsertContracts merges government contracts.
func (s *Service) BatchInsertContracts(ctx context.Context, contracts []domain.PublicContract) error {
	if len(contracts) == 0 {
		return nil
	}

	rows := make([]map[string]any, len(contracts))
	for i, c := range contracts {
		rows[i] = map[string]any{
			"id_contract":     c.IDContract,
			"contract_number": c.ContractNumber,
			"year":            c.Year,
			"agency_ug":       c.AgencyUG,
			"agency_name":     c.AgencyName,
			"contractor_cnpj": c.ContractorCNPJ,
			"contractor_name": c.ContractorName,
			"initial_amount":  c.InitialAmount,
			"final_amount":    c.FinalAmount,
			"signature_date":  c.SignatureDate,
			"end_date":        c.EndDate,
			"object":          c.ObjectSummary,
		}
	}

	query := `
		UNWIND $batch AS row
		MERGE (o:PublicAgency {ug: row.agency_ug})
		ON CREATE SET o.name = row.agency_name

		MERGE (comp:Company {cnpj: row.contractor_cnpj})
		ON CREATE SET comp.legal_name = row.contractor_name

		MERGE (cp:PublicContract {id_contract: row.id_contract})
		ON CREATE SET
			cp.contract_number = row.contract_number,
			cp.year = row.year,
			cp.initial_amount = row.initial_amount,
			cp.final_amount = row.final_amount,
			cp.signature_date = row.signature_date,
			cp.end_date = row.end_date,
			cp.object = row.object

		MERGE (cp)-[:CELEBRADO_POR]->(o)
		MERGE (comp)-[:ASSINOU_CONTRATO]->(cp)
	`

	return s.executeWriteBatch(ctx, query, rows)
}

func (s *Service) executeWriteBatch(ctx context.Context, cypherQuery string, batch []map[string]any) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypherQuery, map[string]any{"batch": batch})
		if err != nil {
			return nil, err
		}
		return res.Consume(ctx)
	})

	return err
}
