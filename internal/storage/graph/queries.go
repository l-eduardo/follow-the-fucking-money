package graph

import (
	"context"
	"fmt"

	"github.com/l-eduardo/follow-the-fucking-money/internal/domain"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// FindMoneyTrail finds the shortest path between two entities in the graph.
func (s *Service) FindMoneyTrail(ctx context.Context, from, to string, maxDepth int, minAmount float64) (*domain.TrailResult, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	cypher := fmt.Sprintf(`
		MATCH (start) WHERE start.cnpj = $from OR start.id = $from OR start.ballot_name =~ ('(?i).*' + $from + '.*') OR start.legal_name =~ ('(?i).*' + $from + '.*')
		MATCH (target) WHERE target.cnpj = $to OR target.id = $to OR target.ballot_name =~ ('(?i).*' + $to + '.*') OR target.legal_name =~ ('(?i).*' + $to + '.*')
		MATCH p = shortestPath((start)-[*..%d]-(target))
		RETURN p
		LIMIT 1
	`, maxDepth)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, map[string]any{"from": from, "to": to})
		if err != nil {
			return nil, err
		}

		if res.Next(ctx) {
			pathRecord, ok := res.Record().Get("p")
			if !ok {
				return nil, fmt.Errorf("no path returned")
			}
			path := pathRecord.(neo4j.Path)
			hops := make([]domain.TrailHop, len(path.Relationships))

			for i, rel := range path.Relationships {
				startNode := path.Nodes[i]
				endNode := path.Nodes[i+1]

				hops[i] = domain.TrailHop{
					FromLabel:    startNode.Labels[0],
					FromID:       fmt.Sprintf("%v", startNode.Props["cnpj"]),
					FromName:     fmt.Sprintf("%v", startNode.Props["ballot_name"]),
					Relationship: rel.Type,
					Properties:   rel.Props,
					ToLabel:      endNode.Labels[0],
					ToID:         fmt.Sprintf("%v", endNode.Props["cnpj"]),
					ToName:       fmt.Sprintf("%v", endNode.Props["ballot_name"]),
				}
			}

			return &domain.TrailResult{
				FromQuery: from,
				ToQuery:   to,
				HopsCount: len(hops),
				Path:      hops,
			}, nil
		}

		return nil, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return &domain.TrailResult{FromQuery: from, ToQuery: to, HopsCount: 0, Path: []domain.TrailHop{}}, nil
	}
	return result.(*domain.TrailResult), nil
}

// FindQuidProQuo queries instances of donors whose companies won contracts after the election.
func (s *Service) FindQuidProQuo(ctx context.Context, electionYear int, minContracts float64) ([]map[string]any, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	cypher := `
		MATCH (donor:Person)-[d:DOOU_PARA]->(cand:Candidate)
		WHERE cand.year = $year AND cand.status IN ['ELEITO', 'ELEITO POR QP']
		MATCH (donor)-[:SOCIO_DE]->(comp:Company)
		MATCH (comp)-[:ASSINOU_CONTRATO]->(cp:PublicContract)-[:CELEBRADO_POR]->(agency:PublicAgency)
		WHERE cp.final_amount >= $minAmount
		WITH cand, donor, comp, agency, sum(d.amount) AS total_donated, sum(cp.final_amount) AS total_contracts
		RETURN cand.ballot_name AS candidate,
		       donor.name AS donor,
		       comp.legal_name AS company,
		       comp.cnpj AS cnpj,
		       agency.name AS agency,
		       total_donated AS donated,
		       total_contracts AS contracted,
		       (total_contracts / total_donated) AS multiplier
		ORDER BY total_contracts DESC
		LIMIT 50
	`

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cursor, err := tx.Run(ctx, cypher, map[string]any{
			"year":      electionYear,
			"minAmount": minContracts,
		})
		if err != nil {
			return nil, err
		}

		var results []map[string]any
		for cursor.Next(ctx) {
			results = append(results, cursor.Record().AsMap())
		}
		return results, cursor.Err()
	})

	if err != nil {
		return nil, err
	}
	return res.([]map[string]any), nil
}

// FindGhostSuppliers detects newly formed companies receiving significant campaign expenditures.
func (s *Service) FindGhostSuppliers(ctx context.Context, electionYear int, minExpense float64) ([]map[string]any, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	cypher := `
		MATCH (comp:Company)-[f:FORNECEU_PARA]->(cand:Candidate)
		WHERE f.year = $year
		WITH comp, cand, sum(f.amount) AS total_received
		WHERE total_received >= $minExpense AND comp.share_capital <= 10000.0
		RETURN comp.cnpj AS cnpj,
		       comp.legal_name AS company,
		       comp.share_capital AS capital,
		       comp.opening_date AS opening_date,
		       cand.ballot_name AS candidate,
		       cand.party AS party,
		       total_received AS total_received
		ORDER BY total_received DESC
		LIMIT 50
	`

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cursor, err := tx.Run(ctx, cypher, map[string]any{
			"year":       electionYear,
			"minExpense": minExpense,
		})
		if err != nil {
			return nil, err
		}

		var results []map[string]any
		for cursor.Next(ctx) {
			results = append(results, cursor.Record().AsMap())
		}
		return results, cursor.Err()
	})

	if err != nil {
		return nil, err
	}
	return res.([]map[string]any), nil
}

// FindAmendmentTriangulation investigates parliamentary amendments redirected to companies of campaign donors.
func (s *Service) FindAmendmentTriangulation(ctx context.Context, year int) ([]map[string]any, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	cypher := `
		MATCH (cand:Candidate)-[:DESTINOU_EMENDA]->(emenda:ParliamentaryAmendment)-[:RECEBIDA_POR]->(agency:PublicAgency)
		MATCH (comp:Company)-[:ASSINOU_CONTRATO]->(cp:PublicContract)-[:CELEBRADO_POR]->(agency)
		MATCH (donor:Person)-[:SOCIO_DE]->(comp)
		MATCH (donor)-[:DOOU_PARA]->(cand)
		RETURN cand.ballot_name AS parliamentarian,
		       agency.name AS municipality,
		       comp.legal_name AS contractor,
		       donor.name AS donor_partner,
		       cp.final_amount AS contract_amount
		LIMIT 50
	`

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cursor, err := tx.Run(ctx, cypher, nil)
		if err != nil {
			return nil, err
		}

		var results []map[string]any
		for cursor.Next(ctx) {
			results = append(results, cursor.Record().AsMap())
		}
		return results, cursor.Err()
	})

	if err != nil {
		return nil, err
	}
	return res.([]map[string]any), nil
}

// GetEntityNeighbors fetches the 1-hop subgraph around a given entity.
func (s *Service) GetEntityNeighbors(ctx context.Context, entityID string, maxHops int) (map[string]any, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	cypher := `
		MATCH (n) WHERE n.cnpj = $id OR n.id = $id OR n.sq_candidate = $id
		OPTIONAL MATCH (n)-[r]-(neighbor)
		RETURN n, collect(r) AS relationships, collect(neighbor) AS neighbors
		LIMIT 1
	`

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cursor, err := tx.Run(ctx, cypher, map[string]any{"id": entityID})
		if err != nil {
			return nil, err
		}

		if cursor.Next(ctx) {
			return cursor.Record().AsMap(), nil
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}
	if res == nil {
		return map[string]any{}, nil
	}
	return res.(map[string]any), nil
}

// GetDatabaseStats returns total counts of nodes and relationships.
func (s *Service) GetDatabaseStats(ctx context.Context) (map[string]any, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: s.dbName,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	cypher := `
		MATCH (n)
		RETURN labels(n)[0] AS label, count(n) AS count
		ORDER BY count DESC
	`

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cursor, err := tx.Run(ctx, cypher, nil)
		if err != nil {
			return nil, err
		}

		stats := make(map[string]any)
		for cursor.Next(ctx) {
			rec := cursor.Record()
			label, _ := rec.Get("label")
			count, _ := rec.Get("count")
			if label != nil {
				stats[fmt.Sprintf("%v", label)] = count
			}
		}
		return stats, cursor.Err()
	})

	if err != nil {
		return nil, err
	}
	return res.(map[string]any), nil
}
