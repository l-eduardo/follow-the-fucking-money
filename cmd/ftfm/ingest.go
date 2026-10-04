package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/l-eduardo/follow-the-fucking-money/internal/domain"
	"github.com/l-eduardo/follow-the-fucking-money/internal/pipeline/extractor"
	"github.com/l-eduardo/follow-the-fucking-money/internal/pipeline/worker"
	"github.com/l-eduardo/follow-the-fucking-money/internal/storage/graph"
	"github.com/l-eduardo/follow-the-fucking-money/pkg/streamutil"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	tseYear      int
	sourcePath   string
	batchSize    int
	workersCount int
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingestão e processamento em lote de dados públicos para o grafo",
}

var ingestTSECmd = &cobra.Command{
	Use:   "tse",
	Short: "Ingere dados eleitorais do TSE (candidaturas, receitas e despesas)",
	Example: `  ftfm ingest tse --year 2022 --source ./downloads/tse/consulta_cand_2022.zip
  ftfm ingest tse --year 2022 --source ./downloads/tse/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		svc, err := graph.NewService(cfg.GraphURI, cfg.GraphUser, cfg.GraphPass, cfg.DatabaseName)
		if err != nil {
			return fmt.Errorf("falha ao conectar no Graph DB: %w", err)
		}
		defer svc.Close(ctx)

		log.Info().
			Int("ano", tseYear).
			Str("origem", sourcePath).
			Int("batch_size", batchSize).
			Msg("Iniciando ingestão do TSE")

		start := time.Now()
		totalProcessed := 0

		// Identify if target is a file or directory
		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return fmt.Errorf("caminho de origem não encontrado: %w", err)
		}

		var filesToProcess []string
		if fileInfo.IsDir() {
			entries, _ := os.ReadDir(sourcePath)
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".zip") || strings.HasSuffix(e.Name(), ".csv") {
					filesToProcess = append(filesToProcess, filepath.Join(sourcePath, e.Name()))
				}
			}
		} else {
			filesToProcess = append(filesToProcess, sourcePath)
		}

		for _, filePath := range filesToProcess {
			fileName := filepath.Base(filePath)
			log.Info().Str("arquivo", fileName).Msg("Processando arquivo do TSE")

			if strings.Contains(fileName, "consulta_cand") {
				// Process Candidates
				pool := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.Candidate, error) {
					return extractor.ParseTSECandidate(row, tseYear)
				})

				batchChan := pool.Start(ctx)
				var writerWG sync.WaitGroup
				writerWG.Add(1)

				// Writer loop
				go func() {
					defer writerWG.Done()
					for batch := range batchChan {
						if err := svc.BatchInsertCandidates(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo candidatos")
						} else {
							totalProcessed += len(batch)
						}
					}
				}()

				_ = streamutil.StreamZipCSV(ctx, filePath, ';', true, func(record []string) error {
					pool.Submit(record)
					return nil
				})
				pool.Close()
				writerWG.Wait()

			} else if strings.Contains(fileName, "receitas") {
				// Process Donations
				pool := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.Donation, error) {
					return extractor.ParseTSEDonation(row, tseYear)
				})

				batchChan := pool.Start(ctx)
				var writerWG sync.WaitGroup
				writerWG.Add(1)

				go func() {
					defer writerWG.Done()
					for batch := range batchChan {
						if err := svc.BatchInsertDonations(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo doações")
						} else {
							totalProcessed += len(batch)
						}
					}
				}()

				_ = streamutil.StreamZipCSV(ctx, filePath, ';', true, func(record []string) error {
					pool.Submit(record)
					return nil
				})
				pool.Close()
				writerWG.Wait()

			} else if strings.Contains(fileName, "despesas") {
				// Process Expenses
				pool := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.Expense, error) {
					return extractor.ParseTSEExpense(row, tseYear)
				})

				batchChan := pool.Start(ctx)
				var writerWG sync.WaitGroup
				writerWG.Add(1)

				go func() {
					defer writerWG.Done()
					for batch := range batchChan {
						if err := svc.BatchInsertExpenses(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo despesas")
						} else {
							totalProcessed += len(batch)
						}
					}
				}()

				_ = streamutil.StreamZipCSV(ctx, filePath, ';', true, func(record []string) error {
					pool.Submit(record)
					return nil
				})
				pool.Close()
				writerWG.Wait()

			} else if strings.Contains(fileName, "bem_candidato") {
				// Process Declared Assets
				pool := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.CandidateAsset, error) {
					return extractor.ParseTSEAsset(row, tseYear)
				})

				batchChan := pool.Start(ctx)
				var writerWG sync.WaitGroup
				writerWG.Add(1)

				go func() {
					defer writerWG.Done()
					for batch := range batchChan {
						if err := svc.BatchInsertAssets(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo bens de candidatos")
						} else {
							totalProcessed += len(batch)
						}
					}
				}()

				_ = streamutil.StreamZipCSV(ctx, filePath, ';', true, func(record []string) error {
					pool.Submit(record)
					return nil
				})
				pool.Close()
				writerWG.Wait()
			}
		}

		log.Info().
			Int("registros_processados", totalProcessed).
			Dur("duracao", time.Since(start)).
			Msg("Ingestão TSE concluída com sucesso")

		return nil
	},
}

func init() {
	ingestTSECmd.Flags().IntVarP(&tseYear, "year", "y", 2022, "Ano da eleição a processar")
	ingestTSECmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Caminho do arquivo ZIP ou diretório do TSE")
	ingestTSECmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Tamanho do lote para gravação no grafo")
	ingestTSECmd.Flags().IntVar(&workersCount, "workers", 8, "Número de workers concorrentes")
	_ = ingestTSECmd.MarkFlagRequired("source")

	ingestCmd.AddCommand(ingestTSECmd)
	rootCmd.AddCommand(ingestCmd)
}
