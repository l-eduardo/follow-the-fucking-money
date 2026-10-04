package main

import (
	"fmt"
	"os"

	"github.com/l-eduardo/follow-the-fucking-money/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	rootCmd = &cobra.Command{
		Use:   "ftfm",
		Short: "Follow-The-Fucking-Money: Plataforma Investigativa em Grafos de Dados Públicos Brasileiros",
		Long: `Follow-The-Fucking-Money (FTFM) cruza e rastreia o fluxo de capitais
entre campanhas eleitorais (TSE), quadros societários (Receita Federal),
contratos públicos (Portal da Transparência/PNCP) e emendas parlamentares.`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	cfg = config.LoadConfig()

	// Configure Zerolog for pretty console in CLI
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if cfg.LogLevel == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// Execute runs the root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
