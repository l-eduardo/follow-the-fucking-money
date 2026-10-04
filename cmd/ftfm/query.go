package main

import (
	"context"
	"fmt"
	"time"

	"github.com/l-eduardo/follow-the-fucking-money/internal/storage/graph"
	"github.com/spf13/cobra"
)

var (
	queryFrom   string
	queryTo     string
	maxHops     int
	minAmount   float64
	targetYear  int
	entityQuery string
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Consultas analíticas e investigações sobre a teia de relações",
}

var trailCmd = &cobra.Command{
	Use:   "trail",
	Short: "Rastreia o caminho do dinheiro entre duas entidades (ShortestPath)",
	Example: `  ftfm query trail --from "00000000000191" --to "LULA" --max-depth 4
  ftfm query trail --from "JOSE DA SILVA" --to "BOLSONARO"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		svc, err := graph.NewService(cfg.GraphURI, cfg.GraphUser, cfg.GraphPass, cfg.DatabaseName)
		if err != nil {
			return fmt.Errorf("falha ao conectar no Graph DB: %w", err)
		}
		defer svc.Close(ctx)

		fmt.Printf("🔍 Rastreando caminho entre '%s' e '%s' (Profundidade Máx: %d)...\n\n", queryFrom, queryTo, maxHops)

		res, err := svc.FindMoneyTrail(ctx, queryFrom, queryTo, maxHops, minAmount)
		if err != nil {
			return err
		}

		if len(res.Path) == 0 {
			fmt.Println("❌ Nenhum caminho encontrado entre as entidades nos critérios informados.")
			return nil
		}

		fmt.Printf("✅ Caminho encontrado com %d conexões:\n\n", len(res.Path))
		for i, hop := range res.Path {
			fmt.Printf("  [%d] (%s: %s) --[%s]--> (%s: %s)\n",
				i+1, hop.FromLabel, hop.FromName, hop.Relationship, hop.ToLabel, hop.ToName)
		}

		return nil
	},
}

var qpqCmd = &cobra.Command{
	Use:   "quid-pro-quo",
	Short: "Identifica doadores cujas empresas ganharam contratos públicos no mandato",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		svc, err := graph.NewService(cfg.GraphURI, cfg.GraphUser, cfg.GraphPass, cfg.DatabaseName)
		if err != nil {
			return err
		}
		defer svc.Close(ctx)

		fmt.Printf("🔍 Investigando padrão Quid Pro Quo (Eleição: %d, Contratos mínimos: R$ %.2f)...\n\n", targetYear, minAmount)

		results, err := svc.FindQuidProQuo(ctx, targetYear, minAmount)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("Nenhum padrão encontrado para os filtros selecionados.")
			return nil
		}

		for _, row := range results {
			fmt.Printf("  • Candidato: %v | Doador: %v | Empresa: %v\n", row["candidate"], row["donor"], row["company"])
			fmt.Printf("    Doação: R$ %.2f ➔ Contratos Recebidos: R$ %.2f (ROI: %.1fx)\n", row["donated"], row["contracted"], row["multiplier"])
			fmt.Println("    ---------------------------------------------------------")
		}

		return nil
	},
}

var ghostCmd = &cobra.Command{
	Use:   "ghost-suppliers",
	Short: "Detecta fornecedores recém-criados ou de capital baixo com despesas expressivas",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		svc, err := graph.NewService(cfg.GraphURI, cfg.GraphUser, cfg.GraphPass, cfg.DatabaseName)
		if err != nil {
			return err
		}
		defer svc.Close(ctx)

		results, err := svc.FindGhostSuppliers(ctx, targetYear, minAmount)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("Nenhum fornecedor suspeito encontrado.")
			return nil
		}

		for _, row := range results {
			fmt.Printf("  • Empresa: %v (%v) | Capital Social: R$ %v\n", row["company"], row["cnpj"], row["capital"])
			fmt.Printf("    Candidato Pago: %v (%v) | Total Recebido: R$ %.2f\n", row["candidate"], row["party"], row["total_received"])
			fmt.Println("    ---------------------------------------------------------")
		}

		return nil
	},
}

func init() {
	trailCmd.Flags().StringVar(&queryFrom, "from", "", "Identificador da origem (CNPJ, CPF ou Nome)")
	trailCmd.Flags().StringVar(&queryTo, "to", "", "Identificador do destino (Candidato, Empresa)")
	trailCmd.Flags().IntVar(&maxHops, "max-depth", 4, "Profundidade máxima no grafo")
	trailCmd.Flags().Float64Var(&minAmount, "min-amount", 0.0, "Valor mínimo de transação")
	_ = trailCmd.MarkFlagRequired("from")
	_ = trailCmd.MarkFlagRequired("to")

	qpqCmd.Flags().IntVarP(&targetYear, "year", "y", 2022, "Ano eleitoral da campanha")
	qpqCmd.Flags().Float64Var(&minAmount, "min-amount", 50000.0, "Valor mínimo dos contratos")

	ghostCmd.Flags().IntVarP(&targetYear, "year", "y", 2022, "Ano eleitoral da campanha")
	ghostCmd.Flags().Float64Var(&minAmount, "min-expense", 100000.0, "Valor mínimo pago pela campanha")

	queryCmd.AddCommand(trailCmd)
	queryCmd.AddCommand(qpqCmd)
	queryCmd.AddCommand(ghostCmd)
	rootCmd.AddCommand(queryCmd)
}
