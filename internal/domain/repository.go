package domain

import (
	"context"
)

// GraphRepository defines writing and administrative methods for graph databases.
type GraphRepository interface {
	CreateConstraintsAndIndexes(ctx context.Context) error
	BatchInsertCandidates(ctx context.Context, candidates []Candidate) error
	BatchInsertDonations(ctx context.Context, donations []Donation) error
	BatchInsertExpenses(ctx context.Context, expenses []Expense) error
	BatchInsertCompanies(ctx context.Context, companies []Company) error
	BatchInsertPartners(ctx context.Context, partners []map[string]any) error
	BatchInsertContracts(ctx context.Context, contracts []PublicContract) error
	Close(ctx context.Context) error
}

// GraphQueryService defines query methods for investigation and API endpoints.
type GraphQueryService interface {
	FindMoneyTrail(ctx context.Context, from, to string, maxDepth int, minAmount float64) (*TrailResult, error)
	FindQuidProQuo(ctx context.Context, electionYear int, minContracts float64) ([]map[string]any, error)
	FindGhostSuppliers(ctx context.Context, electionYear int, minExpense float64) ([]map[string]any, error)
	FindAmendmentTriangulation(ctx context.Context, year int) ([]map[string]any, error)
	GetEntityNeighbors(ctx context.Context, entityID string, maxHops int) (map[string]any, error)
	GetDatabaseStats(ctx context.Context) (map[string]any, error)
}
