package main

import (
	"context"
	"fmt"
	"time"

	"github.com/l-eduardo/follow-the-fucking-money/internal/storage/graph"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var createIndexes bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Verifica conexão e estatísticas do Graph Database (Memgraph / Neo4j)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Info().Str("uri", cfg.GraphURI).Msg("Conectando ao Graph DB...")

		svc, err := graph.NewService(cfg.GraphURI, cfg.GraphUser, cfg.GraphPass, cfg.DatabaseName)
		if err != nil {
			return fmt.Errorf("falha ao instanciar serviço de grafo: %w", err)
		}
		defer svc.Close(ctx)

		if err := svc.Ping(ctx); err != nil {
			return fmt.Errorf("não foi possível conectar ao banco de grafos: %w", err)
		}

		fmt.Println("Conexão com Graph DB estabelecida com sucesso!")

		if createIndexes {
			fmt.Println("Aplicando constraints e índices de busca...")
			if err := svc.CreateConstraintsAndIndexes(ctx); err != nil {
				return err
			}
			fmt.Println("Constraints e índices criados/verificados com sucesso!")
		}

		stats, err := svc.GetDatabaseStats(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("Não foi possível carregar estatísticas do banco")
		} else {
			fmt.Println("\n📊 Estatísticas Atuais da Base de Dados:")
			if len(stats) == 0 {
				fmt.Println("  (Base de dados vazia - execute 'ftfm ingest' para carregar dados)")
			} else {
				for label, count := range stats {
					fmt.Printf("  • %-20s: %v nós\n", label, count)
				}
			}
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&createIndexes, "create-indexes", false, "Aplica as constraints UNIQUE e índices no grafo")
	rootCmd.AddCommand(statusCmd)
}
