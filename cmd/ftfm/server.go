package main

import (
	"fmt"

	"github.com/l-eduardo/follow-the-fucking-money/internal/api"
	"github.com/l-eduardo/follow-the-fucking-money/internal/storage/graph"
	"github.com/spf13/cobra"
)

var (
	serverPort int
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Inicia o servidor de API HTTP REST e visualização",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := graph.NewService(cfg.GraphURI, cfg.GraphUser, cfg.GraphPass, cfg.DatabaseName)
		if err != nil {
			return fmt.Errorf("falha ao conectar no Graph DB: %w", err)
		}
		defer svc.Close(cmd.Context())

		srv := api.NewServer(svc, serverPort)
		return srv.Start()
	},
}

func init() {
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 8075, "Porta TCP do servidor HTTP")
	rootCmd.AddCommand(serverCmd)
}
